package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/relaysite"
	"github.com/looplj/axonhub/internal/ent/relaysiteannouncement"
	"github.com/looplj/axonhub/internal/ent/relaysiteapikey"
	"github.com/looplj/axonhub/internal/ent/relaysitecheckinlog"
	"github.com/looplj/axonhub/internal/ent/relaysitecredential"
	"github.com/looplj/axonhub/internal/ent/relaysitegroup"
	"github.com/looplj/axonhub/internal/ent/relaysitemodelprice"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/pkg/xerrors"
	"github.com/looplj/axonhub/internal/pkg/xtime"
	"github.com/looplj/axonhub/internal/server/scheduler"
)

const (
	relaySiteAutoCheckinTaskName = "relay-site-auto-checkin"
	relaySiteChannelTag          = "relay-site"
)

type RelaySiteServiceParams struct {
	fx.In

	Ent            *ent.Client
	AdapterFactory *RelaySiteAdapterFactory
	ChannelService *ChannelService
}

type RelaySiteService struct {
	*AbstractService

	adapterFactory *RelaySiteAdapterFactory
	channelService *ChannelService
}

type CreateRelaySiteConfigInput struct {
	Name                   string
	Type                   *relaysite.Type
	BaseURL                string
	Status                 *relaysite.Status
	AutoCheckinEnabled     *bool
	Remark                 *string
	Credential             objects.RelaySiteCredential
	CheckinPageURL         *string
	ExternalCheckinPageURL *string
}

type UpdateRelaySiteConfigInput struct {
	Name                   *string
	BaseURL                *string
	Status                 *relaysite.Status
	AutoCheckinEnabled     *bool
	Remark                 *string
	Credential             *objects.RelaySiteCredential
	CheckinPageURL         *string
	ExternalCheckinPageURL *string
}

type ImportRelaySiteAPIKeyToChannelInput struct {
	Name             string
	Type             channel.Type
	BaseURL          string
	SupportedModels  []string
	DefaultTestModel string
	Tags             []string
	Remark           *string
}

type RelaySiteBatchOperationFailure struct {
	RelaySiteID   int    `json:"relaySiteID"`
	RelaySiteName string `json:"relaySiteName"`
	ErrorMessage  string `json:"errorMessage"`
}

type RelaySiteBatchOperationResult struct {
	TotalCount   int                               `json:"totalCount"`
	SuccessCount int                               `json:"successCount"`
	FailedCount  int                               `json:"failedCount"`
	Failures     []*RelaySiteBatchOperationFailure `json:"failures"`
}

func NewRelaySiteService(params RelaySiteServiceParams) *RelaySiteService {
	return &RelaySiteService{
		AbstractService: &AbstractService{db: params.Ent},
		adapterFactory:  params.AdapterFactory,
		channelService:  params.ChannelService,
	}
}

func (s *RelaySiteService) CreateSite(ctx context.Context, input CreateRelaySiteConfigInput) (*ent.RelaySite, error) {
	siteType := relaysite.TypeNewAPI
	if input.Type != nil {
		siteType = *input.Type
	}
	authType, credential, err := normalizeRelaySiteCredential(siteType, input.Credential)
	if err != nil {
		return nil, err
	}

	var site *ent.RelaySite
	err = s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)

		create := client.RelaySite.Create().
			SetName(input.Name).
			SetType(siteType).
			SetBaseURL(input.BaseURL).
			SetNillableStatus(input.Status).
			SetNillableAutoCheckinEnabled(input.AutoCheckinEnabled).
			SetNillableRemark(input.Remark).
			SetNillableCheckinPageURL(input.CheckinPageURL).
			SetNillableExternalCheckinPageURL(input.ExternalCheckinPageURL)

		var saveErr error
		site, saveErr = create.Save(ctx)
		if saveErr != nil {
			return fmt.Errorf("failed to create relay site: %w", saveErr)
		}

		_, saveErr = client.RelaySiteCredential.Create().
			SetRelaySiteID(site.ID).
			SetAuthType(authType).
			SetCredential(credential).
			Save(ctx)
		if saveErr != nil {
			return fmt.Errorf("failed to create relay site credential: %w", saveErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return site, nil
}

func (s *RelaySiteService) UpdateSite(ctx context.Context, id int, input UpdateRelaySiteConfigInput) (*ent.RelaySite, error) {
	var authType relaysitecredential.AuthType
	var credential objects.RelaySiteCredential
	var err error
	if input.Credential != nil {
		existing, getErr := s.entFromContext(ctx).RelaySite.Get(ctx, id)
		if getErr != nil {
			return nil, fmt.Errorf("failed to get relay site: %w", getErr)
		}
		authType, credential, err = normalizeRelaySiteCredential(existing.Type, *input.Credential)
		if err != nil {
			return nil, err
		}
	}

	var site *ent.RelaySite
	channelStatusSynced := false
	err = s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)

		update := client.RelaySite.UpdateOneID(id).
			SetNillableName(input.Name).
			SetNillableBaseURL(input.BaseURL).
			SetNillableStatus(input.Status).
			SetNillableAutoCheckinEnabled(input.AutoCheckinEnabled).
			SetNillableRemark(input.Remark).
			SetNillableCheckinPageURL(input.CheckinPageURL).
			SetNillableExternalCheckinPageURL(input.ExternalCheckinPageURL)

		var saveErr error
		site, saveErr = update.Save(ctx)
		if saveErr != nil {
			return fmt.Errorf("failed to update relay site: %w", saveErr)
		}

		if input.Credential != nil {
			_, saveErr = client.RelaySiteCredential.Query().
				Where(relaysitecredential.RelaySiteIDEQ(id)).
				Only(ctx)
			if saveErr != nil {
				return fmt.Errorf("failed to get relay site credential: %w", saveErr)
			}

			_, saveErr = client.RelaySiteCredential.Update().
				Where(relaysitecredential.RelaySiteIDEQ(id)).
				SetAuthType(authType).
				SetCredential(credential).
				Save(ctx)
			if saveErr != nil {
				return fmt.Errorf("failed to update relay site credential: %w", saveErr)
			}
		}

		if input.Status != nil {
			if err := s.syncRelaySiteChannelsStatus(ctx, id, *input.Status); err != nil {
				return err
			}
			channelStatusSynced = true
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if channelStatusSynced {
		s.channelService.asyncReloadChannels()
	}

	return site, nil
}

func (s *RelaySiteService) RegisterScheduledTasks(ctx context.Context, sched *scheduler.Scheduler) error {
	return sched.Register(ctx, scheduler.TaskSpec{
		Name:        relaySiteAutoCheckinTaskName,
		Description: "Run relay site auto check-in daily",
		CronExpr:    "17 1 * * *",
		Timezone:    "UTC",
	}, s.runAutoCheckinScheduled)
}

func (s *RelaySiteService) runAutoCheckinScheduled(ctx context.Context) {
	ctx = authz.WithSystemBypass(ctx, relaySiteAutoCheckinTaskName)

	sites, err := s.entFromContext(ctx).RelaySite.Query().
		Where(relaysite.TypeEQ(relaysite.TypeNewAPI), relaysite.StatusEQ(relaysite.StatusEnabled), relaysite.AutoCheckinEnabledEQ(true)).
		All(ctx)
	if err != nil {
		log.Warn(ctx, "failed to load relay sites for auto check-in", log.Cause(err))
		return
	}

	for _, site := range sites {
		logEntry, err := s.CheckinSite(ctx, site.ID)
		if err != nil {
			log.Warn(ctx, "failed to auto check in relay site",
				log.Int("relay_site_id", site.ID),
				log.String("relay_site_name", site.Name),
				log.Cause(err),
			)
			continue
		}
		if logEntry != nil && logEntry.Status == relaysitecheckinlog.StatusFailed {
			log.Warn(ctx, "failed to auto check in relay site",
				log.Int("relay_site_id", site.ID),
				log.String("relay_site_name", site.Name),
				log.Cause(errors.New(relaySiteCheckinLogMessage(logEntry))),
			)
		}
	}
}

func (s *RelaySiteService) DeleteSite(ctx context.Context, id int) (*ent.RelaySite, error) {
	var site *ent.RelaySite
	var deletedChannelIDs []int
	err := s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)

		var err error
		site, err = client.RelaySite.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get relay site: %w", err)
		}

		channelIDs, err := s.relaySiteChannelIDs(ctx, id)
		if err != nil {
			return err
		}
		if len(channelIDs) > 0 {
			if _, err := client.Channel.Delete().Where(channel.IDIn(channelIDs...)).Exec(ctx); err != nil {
				return fmt.Errorf("failed to delete relay site channels: %w", err)
			}
			deletedChannelIDs = channelIDs
		}

		if err := client.RelaySite.DeleteOneID(id).Exec(ctx); err != nil {
			return fmt.Errorf("failed to delete relay site: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(deletedChannelIDs) > 0 {
		for _, channelID := range deletedChannelIDs {
			s.channelService.forgetLimiter(channelID)
		}
		s.channelService.asyncReloadChannels()
	}

	return site, nil
}

func (s *RelaySiteService) SyncAllSites(ctx context.Context) (*RelaySiteBatchOperationResult, error) {
	sites, err := s.listAllSites(ctx)
	if err != nil {
		return nil, err
	}

	return s.executeBatchOperation(ctx, sites, func(ctx context.Context, site *ent.RelaySite) error {
		return s.SyncSite(ctx, site.ID)
	}), nil
}

func (s *RelaySiteService) CheckinAllSites(ctx context.Context) (*RelaySiteBatchOperationResult, error) {
	sites, err := s.entFromContext(ctx).RelaySite.Query().
		Where(relaysite.TypeEQ(relaysite.TypeNewAPI)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list new-api relay sites: %w", err)
	}

	return s.executeBatchOperation(ctx, sites, func(ctx context.Context, site *ent.RelaySite) error {
		logEntry, err := s.CheckinSite(ctx, site.ID)
		if err != nil {
			return err
		}
		if logEntry != nil && logEntry.Status == relaysitecheckinlog.StatusFailed {
			return errors.New(relaySiteCheckinLogMessage(logEntry))
		}
		return nil
	}), nil
}

func newRelaySiteBatchOperationResult(totalCount int) *RelaySiteBatchOperationResult {
	return &RelaySiteBatchOperationResult{
		TotalCount: totalCount,
		Failures:   make([]*RelaySiteBatchOperationFailure, 0),
	}
}

func (r *RelaySiteBatchOperationResult) addFailure(site *ent.RelaySite, err error) {
	r.addFailureMessage(site, err.Error())
}

func (r *RelaySiteBatchOperationResult) addFailureMessage(site *ent.RelaySite, message string) {
	r.FailedCount++
	r.Failures = append(r.Failures, &RelaySiteBatchOperationFailure{
		RelaySiteID:   site.ID,
		RelaySiteName: site.Name,
		ErrorMessage:  message,
	})
}

func (s *RelaySiteService) executeBatchOperation(ctx context.Context, sites []*ent.RelaySite, operation func(context.Context, *ent.RelaySite) error) *RelaySiteBatchOperationResult {
	result := newRelaySiteBatchOperationResult(len(sites))
	if len(sites) == 0 {
		return result
	}

	// Use a buffered channel to control concurrency
	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)
	resultChan := make(chan batchOperationItemResult, len(sites))

	// Launch goroutines for each site
	for _, site := range sites {
		site := site // capture loop variable
		go func() {
			semaphore <- struct{}{}        // acquire semaphore
			defer func() { <-semaphore }() // release semaphore

			// Create a new context for each operation to avoid sharing transaction state
			opCtx := contexts.CopyContextWithoutTransaction(ctx)
			err := operation(opCtx, site)

			resultChan <- batchOperationItemResult{
				site: site,
				err:  err,
			}
		}()
	}

	// Collect results
	for i := 0; i < len(sites); i++ {
		item := <-resultChan
		if item.err != nil {
			result.addFailure(item.site, item.err)
		} else {
			result.SuccessCount++
		}
	}

	close(resultChan)
	return result
}

type batchOperationItemResult struct {
	site *ent.RelaySite
	err  error
}

func (s *RelaySiteService) listAllSites(ctx context.Context) ([]*ent.RelaySite, error) {
	sites, err := s.entFromContext(ctx).RelaySite.Query().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list relay sites: %w", err)
	}

	return sites, nil
}

func (s *RelaySiteService) SyncSite(ctx context.Context, id int) error {
	adapter, err := s.adapterForSite(ctx, id)
	if err != nil {
		return err
	}

	apiKeys, err := adapter.ListAPIKeys(ctx)
	if err != nil {
		return s.recordSyncFailure(ctx, id, err)
	}
	groups, err := adapter.ListGroups(ctx)
	if err != nil {
		return s.recordSyncFailure(ctx, id, err)
	}
	balance, err := adapter.GetBalance(ctx)
	if err != nil && !errors.Is(err, ErrRelaySiteAdapterNotImplemented) {
		return s.recordSyncFailure(ctx, id, err)
	}
	modelPrices, err := adapter.ListModelPrices(ctx)
	if err != nil {
		return s.recordSyncFailure(ctx, id, err)
	}
	announcements, err := adapter.ListAnnouncements(ctx)
	if err != nil {
		return s.recordSyncFailure(ctx, id, err)
	}

	if err = s.persistSyncSnapshots(ctx, id, apiKeys, groups, balance, modelPrices, announcements); err != nil {
		return s.recordSyncFailure(ctx, id, err)
	}

	return nil
}

func (s *RelaySiteService) persistSyncSnapshots(ctx context.Context, id int, apiKeys []RelaySiteAPIKeySnapshot, groups []RelaySiteGroupSnapshot, balance *RelaySiteBalance, modelPrices []RelaySiteModelPriceSnapshot, announcements []RelaySiteAnnouncementSnapshot) error {
	now := xtime.UTCNow()
	return s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)

		seenAPIKeys, err := s.syncAPIKeySnapshots(ctx, client, id, apiKeys, now)
		if err != nil {
			return err
		}
		if err := softDeleteMissingRelaySiteAPIKeys(ctx, client, id, seenAPIKeys); err != nil {
			return err
		}

		seenGroups, err := s.syncGroupSnapshots(ctx, client, id, groups, now)
		if err != nil {
			return err
		}
		if err := softDeleteMissingRelaySiteGroups(ctx, client, id, seenGroups); err != nil {
			return err
		}

		if balance != nil {
			if _, err := client.RelaySiteBalanceSnapshot.Create().
				SetRelaySiteID(id).
				SetBalance(balance.Balance).
				SetUnit(balance.Unit).
				SetPulledAt(now).
				Save(ctx); err != nil {
				return fmt.Errorf("failed to create relay site balance snapshot: %w", err)
			}
		}

		seenModelPrices, err := s.syncModelPriceSnapshots(ctx, client, id, modelPrices, now)
		if err != nil {
			return err
		}
		if err := softDeleteMissingRelaySiteModelPrices(ctx, client, id, seenModelPrices); err != nil {
			return err
		}

		seenAnnouncements, err := s.syncAnnouncementSnapshots(ctx, client, id, announcements, now)
		if err != nil {
			return err
		}
		if err := deleteMissingRelaySiteAnnouncements(ctx, client, id, seenAnnouncements); err != nil {
			return err
		}

		_, err = client.RelaySite.UpdateOneID(id).
			SetLastSyncedAt(now).
			ClearLastSyncError().
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update relay site sync status: %w", err)
		}

		return nil
	})
}

func (s *RelaySiteService) syncAPIKeySnapshots(ctx context.Context, client *ent.Client, siteID int, snapshots []RelaySiteAPIKeySnapshot, syncedAt time.Time) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.RemoteID == "" {
			return nil, fmt.Errorf("relay site api key snapshot requires remote id")
		}

		seen[snapshot.RemoteID] = struct{}{}
		status := normalizeRelaySiteAPIKeyStatus(snapshot.Status)
		update := client.RelaySiteAPIKey.Update().
			Where(relaysiteapikey.RelaySiteIDEQ(siteID), relaysiteapikey.RemoteIDEQ(snapshot.RemoteID)).
			SetStatus(status).
			SetMetadata(snapshot.Metadata).
			SetSyncedAt(syncedAt)

		if snapshot.Name == "" {
			update.ClearName()
		} else {
			update.SetName(snapshot.Name)
		}
		if snapshot.GroupName == nil {
			update.ClearGroupName()
		} else {
			update.SetGroupName(*snapshot.GroupName)
		}
		if snapshot.Quota == nil {
			update.ClearQuota()
		} else {
			update.SetQuota(*snapshot.Quota)
		}
		if snapshot.UsedQuota == nil {
			update.ClearUsedQuota()
		} else {
			update.SetUsedQuota(*snapshot.UsedQuota)
		}
		if snapshot.ExpiresAt == nil {
			update.ClearExpiresAt()
		} else {
			update.SetExpiresAt(*snapshot.ExpiresAt)
		}

		affected, err := update.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update relay site api key snapshot %q: %w", snapshot.RemoteID, err)
		}
		if affected > 0 {
			continue
		}

		create := client.RelaySiteAPIKey.Create().
			SetRelaySiteID(siteID).
			SetRemoteID(snapshot.RemoteID).
			SetStatus(status).
			SetNillableGroupName(snapshot.GroupName).
			SetNillableQuota(snapshot.Quota).
			SetNillableUsedQuota(snapshot.UsedQuota).
			SetNillableExpiresAt(snapshot.ExpiresAt).
			SetMetadata(snapshot.Metadata).
			SetSyncedAt(syncedAt)
		if snapshot.Name != "" {
			create.SetName(snapshot.Name)
		}
		if _, err := create.Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to create relay site api key snapshot %q: %w", snapshot.RemoteID, err)
		}
	}

	return seen, nil
}

func (s *RelaySiteService) syncGroupSnapshots(ctx context.Context, client *ent.Client, siteID int, snapshots []RelaySiteGroupSnapshot, syncedAt time.Time) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Name == "" {
			return nil, fmt.Errorf("relay site group snapshot requires name")
		}

		seen[snapshot.Name] = struct{}{}
		update := client.RelaySiteGroup.Update().
			Where(relaysitegroup.RelaySiteIDEQ(siteID), relaysitegroup.NameEQ(snapshot.Name)).
			SetSettings(snapshot.Settings).
			SetSyncedAt(syncedAt)
		if snapshot.Ratio == nil {
			update.ClearRatio()
		} else {
			update.SetRatio(*snapshot.Ratio)
		}

		affected, err := update.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update relay site group snapshot %q: %w", snapshot.Name, err)
		}
		if affected > 0 {
			continue
		}

		_, err = client.RelaySiteGroup.Create().
			SetRelaySiteID(siteID).
			SetName(snapshot.Name).
			SetNillableRatio(snapshot.Ratio).
			SetSettings(snapshot.Settings).
			SetSyncedAt(syncedAt).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create relay site group snapshot %q: %w", snapshot.Name, err)
		}
	}

	return seen, nil
}

func (s *RelaySiteService) syncModelPriceSnapshots(ctx context.Context, client *ent.Client, siteID int, snapshots []RelaySiteModelPriceSnapshot, syncedAt time.Time) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.ModelID == "" {
			return nil, fmt.Errorf("relay site model price snapshot requires model id")
		}

		seen[snapshot.ModelID] = struct{}{}
		update := client.RelaySiteModelPrice.Update().
			Where(relaysitemodelprice.RelaySiteIDEQ(siteID), relaysitemodelprice.ModelIDEQ(snapshot.ModelID)).
			SetPrice(snapshot.Price).
			SetSyncedAt(syncedAt)

		affected, err := update.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update relay site model price snapshot %q: %w", snapshot.ModelID, err)
		}
		if affected > 0 {
			continue
		}

		_, err = client.RelaySiteModelPrice.Create().
			SetRelaySiteID(siteID).
			SetModelID(snapshot.ModelID).
			SetPrice(snapshot.Price).
			SetSyncedAt(syncedAt).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create relay site model price snapshot %q: %w", snapshot.ModelID, err)
		}
	}

	return seen, nil
}

func (s *RelaySiteService) syncAnnouncementSnapshots(ctx context.Context, client *ent.Client, siteID int, snapshots []RelaySiteAnnouncementSnapshot, fetchedAt time.Time) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.RemoteID == "" {
			return nil, fmt.Errorf("relay site announcement snapshot requires remote id")
		}
		content := strings.TrimSpace(snapshot.Content)
		if content == "" {
			continue
		}

		seen[snapshot.RemoteID] = struct{}{}
		contentHash := relaySiteAnnouncementHash(snapshot)
		existing, err := client.RelaySiteAnnouncement.Query().
			Where(relaysiteannouncement.RelaySiteIDEQ(siteID), relaysiteannouncement.RemoteIDEQ(snapshot.RemoteID)).
			Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get relay site announcement %q: %w", snapshot.RemoteID, err)
		}

		if existing != nil {
			update := client.RelaySiteAnnouncement.UpdateOneID(existing.ID).
				SetContent(content).
				SetContentHash(contentHash).
				SetFetchedAt(fetchedAt).
				SetMetadata(snapshot.Metadata)
			if snapshot.Type == nil {
				update.ClearType()
			} else {
				update.SetType(*snapshot.Type)
			}
			if snapshot.Extra == nil {
				update.ClearExtra()
			} else {
				update.SetExtra(*snapshot.Extra)
			}
			if snapshot.PublishedAt == nil {
				update.ClearPublishedAt()
			} else {
				update.SetPublishedAt(*snapshot.PublishedAt)
			}
			if existing.ContentHash != contentHash {
				update.ClearReadAt()
			}
			if _, err := update.Save(ctx); err != nil {
				return nil, fmt.Errorf("failed to update relay site announcement %q: %w", snapshot.RemoteID, err)
			}
			continue
		}

		create := client.RelaySiteAnnouncement.Create().
			SetRelaySiteID(siteID).
			SetRemoteID(snapshot.RemoteID).
			SetContent(content).
			SetContentHash(contentHash).
			SetFetchedAt(fetchedAt).
			SetMetadata(snapshot.Metadata).
			SetNillableType(snapshot.Type).
			SetNillableExtra(snapshot.Extra).
			SetNillablePublishedAt(snapshot.PublishedAt)
		if _, err := create.Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to create relay site announcement %q: %w", snapshot.RemoteID, err)
		}
	}

	return seen, nil
}

func softDeleteMissingRelaySiteAPIKeys(ctx context.Context, client *ent.Client, siteID int, seen map[string]struct{}) error {
	query := client.RelaySiteAPIKey.Delete().Where(relaysiteapikey.RelaySiteIDEQ(siteID))
	if len(seen) > 0 {
		query.Where(relaysiteapikey.RemoteIDNotIn(lo.Keys(seen)...))
	}
	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete stale relay site api key snapshots: %w", err)
	}
	return nil
}

func softDeleteMissingRelaySiteGroups(ctx context.Context, client *ent.Client, siteID int, seen map[string]struct{}) error {
	query := client.RelaySiteGroup.Delete().Where(relaysitegroup.RelaySiteIDEQ(siteID))
	if len(seen) > 0 {
		query.Where(relaysitegroup.NameNotIn(lo.Keys(seen)...))
	}
	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete stale relay site group snapshots: %w", err)
	}
	return nil
}

func softDeleteMissingRelaySiteModelPrices(ctx context.Context, client *ent.Client, siteID int, seen map[string]struct{}) error {
	query := client.RelaySiteModelPrice.Delete().Where(relaysitemodelprice.RelaySiteIDEQ(siteID))
	if len(seen) > 0 {
		query.Where(relaysitemodelprice.ModelIDNotIn(lo.Keys(seen)...))
	}
	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete stale relay site model price snapshots: %w", err)
	}
	return nil
}

func deleteMissingRelaySiteAnnouncements(ctx context.Context, client *ent.Client, siteID int, seen map[string]struct{}) error {
	query := client.RelaySiteAnnouncement.Delete().Where(relaysiteannouncement.RelaySiteIDEQ(siteID))
	if len(seen) > 0 {
		query.Where(relaysiteannouncement.RemoteIDNotIn(lo.Keys(seen)...))
	}
	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete stale relay site announcements: %w", err)
	}
	return nil
}

func relaySiteAnnouncementHash(snapshot RelaySiteAnnouncementSnapshot) string {
	parts := []string{snapshot.RemoteID, snapshot.Content}
	if snapshot.Type != nil {
		parts = append(parts, *snapshot.Type)
	}
	if snapshot.Extra != nil {
		parts = append(parts, *snapshot.Extra)
	}
	if snapshot.PublishedAt != nil {
		parts = append(parts, snapshot.PublishedAt.UTC().Format(time.RFC3339Nano))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func normalizeRelaySiteAPIKeyStatus(status string) relaysiteapikey.Status {
	switch status {
	case relaysiteapikey.StatusEnabled.String():
		return relaysiteapikey.StatusEnabled
	case relaysiteapikey.StatusDisabled.String():
		return relaysiteapikey.StatusDisabled
	default:
		return relaysiteapikey.StatusUnknown
	}
}

func (s *RelaySiteService) CheckinSite(ctx context.Context, id int) (*ent.RelaySiteCheckinLog, error) {
	site, err := s.entFromContext(ctx).RelaySite.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site: %w", err)
	}
	if site.Type != relaysite.TypeNewAPI {
		return nil, fmt.Errorf("relay site type %s does not support checkin", site.Type)
	}

	adapter, err := s.adapterForSite(ctx, id)
	if err != nil {
		return nil, err
	}

	result, err := adapter.Checkin(ctx)
	now := xtime.UTCNow()

	if err != nil {
		status, message := relaySiteCheckinFailureStatusAndMessage(err)
		return s.recordRelaySiteCheckinResult(ctx, id, now, status, message)
	}

	message := "success"
	if result != nil && result.Message != "" {
		message = result.Message
	}

	return s.recordRelaySiteCheckinResult(ctx, id, now, relaysitecheckinlog.StatusSuccess, message)
}

func (s *RelaySiteService) recordRelaySiteCheckinResult(ctx context.Context, id int, executedAt time.Time, status relaysitecheckinlog.Status, message string) (*ent.RelaySiteCheckinLog, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "checkin failed"
	}

	var logEntry *ent.RelaySiteCheckinLog
	err := s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)
		create := client.RelaySiteCheckinLog.Create().
			SetRelaySiteID(id).
			SetExecutedAt(executedAt).
			SetStatus(status)
		if status == relaysitecheckinlog.StatusFailed {
			create.SetErrorMessage(message)
		} else {
			create.SetMessage(message)
		}

		var saveErr error
		logEntry, saveErr = create.Save(ctx)
		if saveErr != nil {
			return fmt.Errorf("failed to record relay site checkin result: %w", saveErr)
		}

		_, saveErr = client.RelaySite.UpdateOneID(id).
			SetLastCheckinAt(executedAt).
			SetLastCheckinResult(message).
			Save(ctx)
		if saveErr != nil {
			return fmt.Errorf("failed to update relay site checkin status: %w", saveErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return logEntry, nil
}

func relaySiteCheckinFailureStatusAndMessage(err error) (relaysitecheckinlog.Status, string) {
	var checkinFailure *relaySiteCheckinFailure
	if errors.As(err, &checkinFailure) {
		if checkinFailure.skipped {
			return relaysitecheckinlog.StatusSkipped, checkinFailure.message
		}
		return relaysitecheckinlog.StatusFailed, checkinFailure.message
	}

	if codedErr, ok := xerrors.IsCodedError(err); ok {
		if status, ok := codedErr.Extensions["status"].(string); ok && strings.TrimSpace(status) != "" {
			return relaysitecheckinlog.StatusFailed, "status " + strings.TrimSpace(status)
		}
		if detail, ok := codedErr.Extensions["detail"].(string); ok && strings.TrimSpace(detail) != "" {
			return relaysitecheckinlog.StatusFailed, normalizeRelaySiteCheckinMessage(detail)
		}
	}

	return relaysitecheckinlog.StatusFailed, normalizeRelaySiteCheckinMessage(err.Error())
}

func normalizeRelaySiteCheckinMessage(message string) string {
	message = strings.TrimSpace(message)
	prefixes := []string{
		"relay site upstream request failed:",
		"new-api checkin failed:",
		"new-api POST /api/user/checkin failed:",
	}
	for {
		previous := message
		for _, prefix := range prefixes {
			if strings.HasPrefix(message, prefix) {
				message = strings.TrimSpace(strings.TrimPrefix(message, prefix))
			}
		}
		if message == previous {
			break
		}
	}
	if status := relaySiteHTTPStatusMessage(message); status != "" {
		return status
	}
	if message == "" {
		return "checkin failed"
	}
	return message
}

func relaySiteHTTPStatusMessage(message string) string {
	const marker = " with status "
	idx := strings.LastIndex(message, marker)
	if idx < 0 {
		return ""
	}
	status := strings.TrimSpace(message[idx+len(marker):])
	if status == "" {
		return ""
	}
	return "status " + status
}

func isRelaySiteCheckinSkippedMessage(message string) bool {
	switch strings.TrimSpace(message) {
	case "今日已签到", "签到功能未启用":
		return true
	default:
		return false
	}
}

func relaySiteCheckinLogMessage(logEntry *ent.RelaySiteCheckinLog) string {
	if logEntry == nil {
		return "checkin failed"
	}
	if logEntry.ErrorMessage != nil && strings.TrimSpace(*logEntry.ErrorMessage) != "" {
		return strings.TrimSpace(*logEntry.ErrorMessage)
	}
	if logEntry.Message != nil && strings.TrimSpace(*logEntry.Message) != "" {
		return strings.TrimSpace(*logEntry.Message)
	}
	return "checkin failed"
}

func (s *RelaySiteService) RefreshAnnouncements(ctx context.Context, id int) (*ent.RelaySite, error) {
	adapter, err := s.adapterForSite(ctx, id)
	if err != nil {
		return nil, err
	}

	announcements, err := adapter.ListAnnouncements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list relay site announcements: %w", err)
	}

	fetchedAt := xtime.UTCNow()
	if err := s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)
		seen, err := s.syncAnnouncementSnapshots(ctx, client, id, announcements, fetchedAt)
		if err != nil {
			return err
		}
		return deleteMissingRelaySiteAnnouncements(ctx, client, id, seen)
	}); err != nil {
		return nil, err
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, id)
}

func (s *RelaySiteService) MarkAnnouncementsRead(ctx context.Context, relaySiteID int) (*ent.RelaySite, error) {
	readAt := xtime.UTCNow()
	if _, err := s.entFromContext(ctx).RelaySiteAnnouncement.Update().
		Where(relaysiteannouncement.RelaySiteIDEQ(relaySiteID), relaysiteannouncement.ReadAtIsNil()).
		SetReadAt(readAt).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("failed to mark relay site announcements read: %w", err)
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, relaySiteID)
}

func (s *RelaySiteService) ImportAPIKeyToChannel(ctx context.Context, relaySiteAPIKeyID int, input ImportRelaySiteAPIKeyToChannelInput) (*ent.Channel, error) {
	apiKey, err := s.entFromContext(ctx).RelaySiteAPIKey.Get(ctx, relaySiteAPIKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site api key: %w", err)
	}

	fullKey := relaySiteAPIKeyValueFromMetadata(apiKey.Metadata)
	if fullKey == "" {
		fullKey, err = s.adapterAPIKey(ctx, apiKey.RelaySiteID, apiKey.RemoteID)
		if err != nil {
			return nil, err
		}
	}

	channelInput, err := buildRelaySiteChannelInput(input, fullKey)
	if err != nil {
		return nil, err
	}
	channelInput.Tags = appendRelaySiteChannelTags(channelInput.Tags, apiKey.RelaySiteID, apiKey.ID)

	createdChannel, err := s.channelService.CreateChannel(ctx, channelInput)
	if err != nil {
		return nil, fmt.Errorf("failed to import relay site api key to channel: %w", err)
	}

	// 导入后立即启用 Channel（CreateChannelInput 不支持 Status 字段，需要单独更新）
	err = s.entFromContext(ctx).Channel.UpdateOneID(createdChannel.ID).SetStatus(channel.StatusEnabled).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to enable imported channel: %w", err)
	}
	createdChannel.Status = channel.StatusEnabled

	return createdChannel, nil
}

func (s *RelaySiteService) CreateAPIKey(ctx context.Context, relaySiteID int, input RelaySiteAPIKeyConfigInput) (*ent.RelaySite, error) {
	input, err := normalizeRelaySiteAPIKeyConfigInput(input)
	if err != nil {
		return nil, err
	}

	adapter, err := s.adapterForSite(ctx, relaySiteID)
	if err != nil {
		return nil, err
	}
	if err := adapter.CreateAPIKey(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to create relay site api key: %w", err)
	}
	if err := s.SyncSite(ctx, relaySiteID); err != nil {
		return nil, err
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, relaySiteID)
}

func (s *RelaySiteService) UpdateAPIKey(ctx context.Context, relaySiteAPIKeyID int, input RelaySiteAPIKeyConfigInput) (*ent.RelaySite, error) {
	input, err := normalizeRelaySiteAPIKeyConfigInput(input)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.entFromContext(ctx).RelaySiteAPIKey.Get(ctx, relaySiteAPIKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site api key: %w", err)
	}

	adapter, err := s.adapterForSite(ctx, apiKey.RelaySiteID)
	if err != nil {
		return nil, err
	}
	if err := adapter.UpdateAPIKey(ctx, apiKey.RemoteID, input); err != nil {
		return nil, fmt.Errorf("failed to update relay site api key: %w", err)
	}
	if err := s.SyncSite(ctx, apiKey.RelaySiteID); err != nil {
		return nil, err
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, apiKey.RelaySiteID)
}

func (s *RelaySiteService) DeleteAPIKey(ctx context.Context, relaySiteAPIKeyID int) (*ent.RelaySite, error) {
	apiKey, err := s.entFromContext(ctx).RelaySiteAPIKey.Get(ctx, relaySiteAPIKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site api key: %w", err)
	}

	adapter, err := s.adapterForSite(ctx, apiKey.RelaySiteID)
	if err != nil {
		return nil, err
	}
	if err := adapter.DeleteAPIKey(ctx, apiKey.RemoteID); err != nil {
		return nil, fmt.Errorf("failed to delete relay site api key: %w", err)
	}
	if err := s.SyncSite(ctx, apiKey.RelaySiteID); err != nil {
		return nil, err
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, apiKey.RelaySiteID)
}

func (s *RelaySiteService) CreateAPIKeysForAllGroups(ctx context.Context, relaySiteID int, input RelaySiteAPIKeyConfigInput) (*ent.RelaySite, error) {
	input, err := normalizeRelaySiteAPIKeyConfigInput(input)
	if err != nil {
		return nil, err
	}

	groups, err := s.entFromContext(ctx).RelaySiteGroup.Query().
		Where(relaysitegroup.RelaySiteIDEQ(relaySiteID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list relay site groups: %w", err)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("relay site has no synced groups")
	}

	adapter, err := s.adapterForSite(ctx, relaySiteID)
	if err != nil {
		return nil, err
	}

	baseName := input.Name
	for _, group := range groups {
		groupName := group.Name
		groupInput := input
		groupInput.Group = &groupName
		groupInput.Name = fmt.Sprintf("%s-%s", baseName, groupName)
		if err := adapter.CreateAPIKey(ctx, groupInput); err != nil {
			return nil, fmt.Errorf("failed to create relay site api key for group %q: %w", groupName, err)
		}
	}

	if err := s.SyncSite(ctx, relaySiteID); err != nil {
		return nil, err
	}

	return s.entFromContext(ctx).RelaySite.Get(ctx, relaySiteID)
}

func (s *RelaySiteService) adapterAPIKey(ctx context.Context, relaySiteID int, remoteID string) (string, error) {
	adapter, err := s.adapterForSite(ctx, relaySiteID)
	if err != nil {
		return "", err
	}

	key, err := adapter.GetAPIKey(ctx, remoteID)
	if err != nil {
		return "", fmt.Errorf("failed to get relay site api key value: %w", err)
	}

	return key, nil
}

func (s *RelaySiteService) syncRelaySiteChannelsStatus(ctx context.Context, relaySiteID int, status relaysite.Status) error {
	var channelStatus channel.Status
	switch status {
	case relaysite.StatusEnabled:
		channelStatus = channel.StatusEnabled
	case relaysite.StatusDisabled:
		channelStatus = channel.StatusDisabled
	default:
		return nil
	}

	channelIDs, err := s.relaySiteChannelIDs(ctx, relaySiteID)
	if err != nil {
		return err
	}
	if len(channelIDs) == 0 {
		return nil
	}

	if _, err := s.entFromContext(ctx).Channel.Update().
		Where(channel.IDIn(channelIDs...)).
		SetStatus(channelStatus).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to sync relay site channel status: %w", err)
	}

	return nil
}

func (s *RelaySiteService) relaySiteChannelIDs(ctx context.Context, relaySiteID int) ([]int, error) {
	tag := relaySiteScopedChannelTag(relaySiteID)
	channels, err := s.entFromContext(ctx).Channel.Query().
		Where(channel.TagsNotNil()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query relay site channels: %w", err)
	}

	ids := make([]int, 0, len(channels))
	for _, ch := range channels {
		if lo.Contains(ch.Tags, tag) {
			ids = append(ids, ch.ID)
		}
	}

	return ids, nil
}

func appendRelaySiteChannelTags(tags []string, relaySiteID int, relaySiteAPIKeyID int) []string {
	return normalizeStringList(append(tags,
		relaySiteChannelTag,
		relaySiteScopedChannelTag(relaySiteID),
		fmt.Sprintf("relay-site-api-key:%d", relaySiteAPIKeyID),
	))
}

func relaySiteScopedChannelTag(relaySiteID int) string {
	return fmt.Sprintf("relay-site:%d", relaySiteID)
}

func relaySiteAPIKeyValueFromMetadata(metadata objects.RelaySiteAPIKeyMetadata) string {
	key, _ := metadata.Raw["key"].(string)
	key = strings.TrimSpace(key)
	if key == "" || strings.Contains(key, "*") {
		return ""
	}
	return key
}

func normalizeRelaySiteAPIKeyConfigInput(input RelaySiteAPIKeyConfigInput) (RelaySiteAPIKeyConfigInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return input, fmt.Errorf("relay site api key name is required")
	}
	if input.Group != nil {
		group := strings.TrimSpace(*input.Group)
		if group == "" {
			input.Group = nil
		} else {
			input.Group = &group
		}
	}
	if input.ModelLimits != nil {
		modelLimits := strings.TrimSpace(*input.ModelLimits)
		input.ModelLimits = &modelLimits
	}
	if input.AllowIps != nil {
		allowIps := strings.TrimSpace(*input.AllowIps)
		input.AllowIps = &allowIps
	}
	return input, nil
}

func buildRelaySiteChannelInput(input ImportRelaySiteAPIKeyToChannelInput, apiKey string) (ent.CreateChannelInput, error) {
	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimSpace(input.BaseURL)
	defaultTestModel := strings.TrimSpace(input.DefaultTestModel)
	supportedModels := normalizeStringList(input.SupportedModels)
	tags := normalizeStringList(input.Tags)
	if name == "" {
		return ent.CreateChannelInput{}, fmt.Errorf("channel name is required")
	}
	if baseURL == "" {
		return ent.CreateChannelInput{}, fmt.Errorf("channel base url is required")
	}
	if apiKey == "" {
		return ent.CreateChannelInput{}, fmt.Errorf("relay site api key value is required")
	}
	if len(supportedModels) == 0 {
		return ent.CreateChannelInput{}, fmt.Errorf("channel supported models are required")
	}
	if defaultTestModel == "" {
		return ent.CreateChannelInput{}, fmt.Errorf("channel default test model is required")
	}

	return ent.CreateChannelInput{
		Type:                    input.Type,
		BaseURL:                 &baseURL,
		Name:                    name,
		Credentials:             objects.ChannelCredentials{APIKey: apiKey},
		SupportedModels:         supportedModels,
		AutoSyncSupportedModels: lo.ToPtr(true),
		ManualModels:            []string{},
		Tags:                    tags,
		DefaultTestModel:        defaultTestModel,
		Settings: &objects.ChannelSettings{
			ModelMappings: []objects.ModelMapping{},
		},
		Remark: input.Remark,
	}, nil
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *RelaySiteService) adapterForSite(ctx context.Context, id int) (RelaySiteAdapter, error) {
	client := s.entFromContext(ctx)

	site, err := client.RelaySite.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site: %w", err)
	}

	credential, err := client.RelaySiteCredential.Query().
		Where(relaysitecredential.RelaySiteIDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay site credential: %w", err)
	}

	adapter, err := s.adapterFactory.New(site.Type.String(), RelaySiteAdapterConfig{
		BaseURL:    site.BaseURL,
		Credential: credential.Credential,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create relay site adapter: %w", err)
	}

	return adapter, nil
}

func (s *RelaySiteService) recordSyncFailure(ctx context.Context, id int, cause error) error {
	client := s.entFromContext(ctx)
	_, err := client.RelaySite.UpdateOneID(id).
		SetLastSyncedAt(xtime.UTCNow()).
		SetLastSyncError(cause.Error()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update relay site sync failure: %w", err)
	}

	return cause
}

func normalizeRelaySiteCredential(siteType relaysite.Type, input objects.RelaySiteCredential) (relaysitecredential.AuthType, objects.RelaySiteCredential, error) {
	switch input.AuthType {
	case "token":
		if siteType == relaysite.TypeSub2api {
			return "", input, fmt.Errorf("sub2api relay site requires jwt credential")
		}
		if input.Token == "" {
			return "", input, fmt.Errorf("relay site token credential requires token")
		}
		if input.UserID <= 0 {
			return "", input, fmt.Errorf("relay site token credential requires user id")
		}
		input.Username = ""
		input.Password = ""
		input.RefreshToken = ""
		input.TokenExpiresAt = nil
		return relaysitecredential.AuthTypeToken, input, nil
	case "password":
		if siteType == relaysite.TypeSub2api {
			return "", input, fmt.Errorf("sub2api relay site requires jwt credential")
		}
		if input.Username == "" || input.Password == "" {
			return "", input, fmt.Errorf("relay site password credential requires username and password")
		}
		input.Token = ""
		input.UserID = 0
		input.RefreshToken = ""
		input.TokenExpiresAt = nil
		return relaysitecredential.AuthTypePassword, input, nil
	case "jwt":
		if siteType != relaysite.TypeSub2api {
			return "", input, fmt.Errorf("jwt credential is only supported for sub2api relay sites")
		}
		if input.Token == "" {
			return "", input, fmt.Errorf("sub2api jwt credential requires access token")
		}
		input.UserID = 0
		input.Username = ""
		input.Password = ""
		return relaysitecredential.AuthTypeJwt, input, nil
	default:
		return "", input, fmt.Errorf("unsupported relay site credential auth type: %s", input.AuthType)
	}
}
