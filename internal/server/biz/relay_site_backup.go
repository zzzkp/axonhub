package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/relaysite"
	"github.com/looplj/axonhub/internal/ent/relaysiteapikey"
	"github.com/looplj/axonhub/internal/ent/relaysitecheckinlog"
	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/pkg/xtime"
)

const relaySitesBackupVersion = 1

type RelaySitesBackupFile struct {
	Version    int                    `json:"version"`
	ExportedAt time.Time              `json:"exportedAt"`
	Sites      []RelaySitesBackupSite `json:"sites"`
}

type RelaySitesBackupSite struct {
	Name                   string                         `json:"name"`
	Type                   relaysite.Type                 `json:"type"`
	BaseURL                string                         `json:"baseURL"`
	Status                 relaysite.Status               `json:"status"`
	AutoCheckinEnabled     bool                           `json:"autoCheckinEnabled"`
	Remark                 *string                        `json:"remark,omitempty"`
	LastSyncedAt           *time.Time                     `json:"lastSyncedAt,omitempty"`
	LastSyncError          *string                        `json:"lastSyncError,omitempty"`
	LastCheckinAt          *time.Time                     `json:"lastCheckinAt,omitempty"`
	LastCheckinResult      *string                        `json:"lastCheckinResult,omitempty"`
	CheckinPageURL         *string                        `json:"checkinPageURL,omitempty"`
	ExternalCheckinPageURL *string                        `json:"externalCheckinPageURL,omitempty"`
	Credential             objects.RelaySiteCredential    `json:"credential"`
	APIKeys                []RelaySitesBackupAPIKey       `json:"apiKeys,omitempty"`
	Groups                 []RelaySitesBackupGroup        `json:"groups,omitempty"`
	BalanceSnapshots       []RelaySitesBackupBalance      `json:"balanceSnapshots,omitempty"`
	ModelPrices            []RelaySitesBackupModelPrice   `json:"modelPrices,omitempty"`
	CheckinLogs            []RelaySitesBackupCheckinLog   `json:"checkinLogs,omitempty"`
	Announcements          []RelaySitesBackupAnnouncement `json:"announcements,omitempty"`
}

type RelaySitesBackupAPIKey struct {
	RemoteID  string                          `json:"remoteID"`
	Name      string                          `json:"name,omitempty"`
	Status    relaysiteapikey.Status          `json:"status"`
	GroupName *string                         `json:"groupName,omitempty"`
	Quota     *float64                        `json:"quota,omitempty"`
	UsedQuota *float64                        `json:"usedQuota,omitempty"`
	ExpiresAt *time.Time                      `json:"expiresAt,omitempty"`
	Metadata  objects.RelaySiteAPIKeyMetadata `json:"metadata,omitempty"`
	SyncedAt  time.Time                       `json:"syncedAt"`
}

type RelaySitesBackupGroup struct {
	Name     string                         `json:"name"`
	Ratio    *float64                       `json:"ratio,omitempty"`
	Settings objects.RelaySiteGroupSettings `json:"settings,omitempty"`
	SyncedAt time.Time                      `json:"syncedAt"`
}

type RelaySitesBackupBalance struct {
	Balance  float64   `json:"balance"`
	Unit     string    `json:"unit"`
	PulledAt time.Time `json:"pulledAt"`
}

type RelaySitesBackupModelPrice struct {
	ModelID  string                            `json:"modelID"`
	Price    objects.RelaySiteRemoteModelPrice `json:"price,omitempty"`
	SyncedAt time.Time                         `json:"syncedAt"`
}

type RelaySitesBackupCheckinLog struct {
	ExecutedAt   time.Time                  `json:"executedAt"`
	Status       relaysitecheckinlog.Status `json:"status"`
	Message      *string                    `json:"message,omitempty"`
	ErrorMessage *string                    `json:"errorMessage,omitempty"`
}

type RelaySitesBackupAnnouncement struct {
	RemoteID    string                                `json:"remoteID"`
	Type        *string                               `json:"type,omitempty"`
	Content     string                                `json:"content"`
	Extra       *string                               `json:"extra,omitempty"`
	ContentHash string                                `json:"contentHash"`
	PublishedAt *time.Time                            `json:"publishedAt,omitempty"`
	FetchedAt   time.Time                             `json:"fetchedAt"`
	ReadAt      *time.Time                            `json:"readAt,omitempty"`
	Metadata    objects.RelaySiteAnnouncementMetadata `json:"metadata,omitempty"`
}

func (s *RelaySiteService) ExportBackup(ctx context.Context) (string, error) {
	backup := RelaySitesBackupFile{
		Version:    relaySitesBackupVersion,
		ExportedAt: xtime.UTCNow(),
		Sites:      []RelaySitesBackupSite{},
	}

	sites, err := s.entFromContext(ctx).RelaySite.Query().
		Order(ent.Asc(relaysite.FieldID)).
		All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list relay sites for backup: %w", err)
	}

	for _, site := range sites {
		backupSite, err := s.exportBackupSite(ctx, site)
		if err != nil {
			return "", err
		}
		backup.Sites = append(backup.Sites, backupSite)
	}

	payload, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal relay sites backup: %w", err)
	}

	return string(payload), nil
}

func (s *RelaySiteService) exportBackupSite(ctx context.Context, site *ent.RelaySite) (RelaySitesBackupSite, error) {
	credential, err := site.QueryCredential().Only(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to get relay site credential for %q: %w", site.Name, err)
	}

	apiKeys, err := site.QueryAPIKeys().Order(ent.Asc(relaysiteapikey.FieldID)).All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site api keys for %q: %w", site.Name, err)
	}
	groups, err := site.QueryGroups().All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site groups for %q: %w", site.Name, err)
	}
	balances, err := site.QueryBalanceSnapshots().All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site balance snapshots for %q: %w", site.Name, err)
	}
	modelPrices, err := site.QueryModelPrices().All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site model prices for %q: %w", site.Name, err)
	}
	checkinLogs, err := site.QueryCheckinLogs().All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site checkin logs for %q: %w", site.Name, err)
	}
	announcements, err := site.QueryAnnouncements().All(ctx)
	if err != nil {
		return RelaySitesBackupSite{}, fmt.Errorf("failed to list relay site announcements for %q: %w", site.Name, err)
	}

	backupSite := RelaySitesBackupSite{
		Name:                   site.Name,
		Type:                   site.Type,
		BaseURL:                site.BaseURL,
		Status:                 site.Status,
		AutoCheckinEnabled:     site.AutoCheckinEnabled,
		Remark:                 site.Remark,
		LastSyncedAt:           site.LastSyncedAt,
		LastSyncError:          site.LastSyncError,
		LastCheckinAt:          site.LastCheckinAt,
		LastCheckinResult:      site.LastCheckinResult,
		CheckinPageURL:         site.CheckinPageURL,
		ExternalCheckinPageURL: site.ExternalCheckinPageURL,
		Credential:             credential.Credential,
		APIKeys:                make([]RelaySitesBackupAPIKey, 0, len(apiKeys)),
		Groups:                 make([]RelaySitesBackupGroup, 0, len(groups)),
		BalanceSnapshots:       make([]RelaySitesBackupBalance, 0, len(balances)),
		ModelPrices:            make([]RelaySitesBackupModelPrice, 0, len(modelPrices)),
		CheckinLogs:            make([]RelaySitesBackupCheckinLog, 0, len(checkinLogs)),
		Announcements:          make([]RelaySitesBackupAnnouncement, 0, len(announcements)),
	}

	for _, apiKey := range apiKeys {
		backupSite.APIKeys = append(backupSite.APIKeys, RelaySitesBackupAPIKey{
			RemoteID:  apiKey.RemoteID,
			Name:      apiKey.Name,
			Status:    apiKey.Status,
			GroupName: apiKey.GroupName,
			Quota:     apiKey.Quota,
			UsedQuota: apiKey.UsedQuota,
			ExpiresAt: apiKey.ExpiresAt,
			Metadata:  apiKey.Metadata,
			SyncedAt:  apiKey.SyncedAt,
		})
	}
	for _, group := range groups {
		backupSite.Groups = append(backupSite.Groups, RelaySitesBackupGroup{
			Name:     group.Name,
			Ratio:    group.Ratio,
			Settings: group.Settings,
			SyncedAt: group.SyncedAt,
		})
	}
	for _, balance := range balances {
		backupSite.BalanceSnapshots = append(backupSite.BalanceSnapshots, RelaySitesBackupBalance{
			Balance:  balance.Balance,
			Unit:     balance.Unit,
			PulledAt: balance.PulledAt,
		})
	}
	for _, modelPrice := range modelPrices {
		backupSite.ModelPrices = append(backupSite.ModelPrices, RelaySitesBackupModelPrice{
			ModelID:  modelPrice.ModelID,
			Price:    modelPrice.Price,
			SyncedAt: modelPrice.SyncedAt,
		})
	}
	for _, checkinLog := range checkinLogs {
		backupSite.CheckinLogs = append(backupSite.CheckinLogs, RelaySitesBackupCheckinLog{
			ExecutedAt:   checkinLog.ExecutedAt,
			Status:       checkinLog.Status,
			Message:      checkinLog.Message,
			ErrorMessage: checkinLog.ErrorMessage,
		})
	}
	for _, announcement := range announcements {
		backupSite.Announcements = append(backupSite.Announcements, RelaySitesBackupAnnouncement{
			RemoteID:    announcement.RemoteID,
			Type:        announcement.Type,
			Content:     announcement.Content,
			Extra:       announcement.Extra,
			ContentHash: announcement.ContentHash,
			PublishedAt: announcement.PublishedAt,
			FetchedAt:   announcement.FetchedAt,
			ReadAt:      announcement.ReadAt,
			Metadata:    announcement.Metadata,
		})
	}

	return backupSite, nil
}

func (s *RelaySiteService) ImportBackup(ctx context.Context, payload string) error {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return fmt.Errorf("relay sites backup payload is required")
	}

	var backup RelaySitesBackupFile
	if err := json.Unmarshal([]byte(payload), &backup); err != nil {
		return fmt.Errorf("failed to parse relay sites backup: %w", err)
	}
	if backup.Version != relaySitesBackupVersion {
		return fmt.Errorf("unsupported relay sites backup version: %d", backup.Version)
	}

	return s.RunInTransaction(ctx, func(ctx context.Context) error {
		client := s.entFromContext(ctx)
		if err := clearRelaySiteBackupData(ctx, client); err != nil {
			return err
		}

		for _, site := range backup.Sites {
			if err := s.importBackupSite(ctx, client, site); err != nil {
				return err
			}
		}

		return nil
	})
}

func clearRelaySiteBackupData(ctx context.Context, client *ent.Client) error {
	hardDeleteCtx := schematype.SkipSoftDelete(ctx)

	if _, err := client.RelaySiteCredential.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site credentials: %w", err)
	}
	if _, err := client.RelaySiteAPIKey.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site api keys: %w", err)
	}
	if _, err := client.RelaySiteGroup.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site groups: %w", err)
	}
	if _, err := client.RelaySiteBalanceSnapshot.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site balance snapshots: %w", err)
	}
	if _, err := client.RelaySiteModelPrice.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site model prices: %w", err)
	}
	if _, err := client.RelaySiteCheckinLog.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site checkin logs: %w", err)
	}
	if _, err := client.RelaySiteAnnouncement.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay site announcements: %w", err)
	}
	if _, err := client.RelaySite.Delete().Exec(hardDeleteCtx); err != nil {
		return fmt.Errorf("failed to clear relay sites: %w", err)
	}

	return nil
}

func (s *RelaySiteService) importBackupSite(ctx context.Context, client *ent.Client, backup RelaySitesBackupSite) error {
	backup.Name = strings.TrimSpace(backup.Name)
	backup.BaseURL = strings.TrimSpace(backup.BaseURL)
	if backup.Name == "" {
		return fmt.Errorf("relay site backup requires name")
	}
	if backup.BaseURL == "" {
		return fmt.Errorf("relay site backup %q requires base url", backup.Name)
	}
	siteType := backup.Type
	if siteType == "" {
		siteType = relaysite.TypeNewAPI
	}
	if siteType != relaysite.TypeNewAPI && siteType != relaysite.TypeSub2api && siteType != relaysite.TypeDoneHub {
		return fmt.Errorf("unsupported relay site type in backup %q: %s", backup.Name, siteType)
	}
	status := backup.Status
	if status == "" {
		status = relaysite.StatusDisabled
	}

	authType, credential, err := normalizeRelaySiteCredential(siteType, backup.Credential)
	if err != nil {
		return fmt.Errorf("invalid relay site credential for %q: %w", backup.Name, err)
	}

	createdSite, err := client.RelaySite.Create().
		SetName(backup.Name).
		SetType(siteType).
		SetBaseURL(backup.BaseURL).
		SetStatus(status).
		SetAutoCheckinEnabled(backup.AutoCheckinEnabled).
		SetNillableRemark(backup.Remark).
		SetNillableLastSyncedAt(backup.LastSyncedAt).
		SetNillableLastSyncError(backup.LastSyncError).
		SetNillableLastCheckinAt(backup.LastCheckinAt).
		SetNillableLastCheckinResult(backup.LastCheckinResult).
		SetNillableCheckinPageURL(backup.CheckinPageURL).
		SetNillableExternalCheckinPageURL(backup.ExternalCheckinPageURL).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create relay site %q from backup: %w", backup.Name, err)
	}

	if _, err := client.RelaySiteCredential.Create().
		SetRelaySiteID(createdSite.ID).
		SetAuthType(authType).
		SetCredential(credential).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to create relay site credential for %q from backup: %w", backup.Name, err)
	}

	if err := importRelaySiteBackupSnapshots(ctx, client, createdSite.ID, backup); err != nil {
		return fmt.Errorf("failed to import relay site snapshots for %q: %w", backup.Name, err)
	}

	return nil
}

func importRelaySiteBackupSnapshots(ctx context.Context, client *ent.Client, siteID int, backup RelaySitesBackupSite) error {
	for _, apiKey := range backup.APIKeys {
		if apiKey.RemoteID == "" {
			return fmt.Errorf("relay site api key backup requires remote id")
		}
		if apiKey.SyncedAt.IsZero() {
			return fmt.Errorf("relay site api key backup %q requires synced at", apiKey.RemoteID)
		}
		status := apiKey.Status
		if status == "" {
			status = relaysiteapikey.StatusUnknown
		}
		create := client.RelaySiteAPIKey.Create().
			SetRelaySiteID(siteID).
			SetRemoteID(apiKey.RemoteID).
			SetStatus(status).
			SetNillableGroupName(apiKey.GroupName).
			SetNillableQuota(apiKey.Quota).
			SetNillableUsedQuota(apiKey.UsedQuota).
			SetNillableExpiresAt(apiKey.ExpiresAt).
			SetMetadata(apiKey.Metadata).
			SetSyncedAt(apiKey.SyncedAt)
		if apiKey.Name != "" {
			create.SetName(apiKey.Name)
		}
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site api key backup %q: %w", apiKey.RemoteID, err)
		}
	}

	for _, group := range backup.Groups {
		if group.Name == "" {
			return fmt.Errorf("relay site group backup requires name")
		}
		if group.SyncedAt.IsZero() {
			return fmt.Errorf("relay site group backup %q requires synced at", group.Name)
		}
		if _, err := client.RelaySiteGroup.Create().
			SetRelaySiteID(siteID).
			SetName(group.Name).
			SetNillableRatio(group.Ratio).
			SetSettings(group.Settings).
			SetSyncedAt(group.SyncedAt).
			Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site group backup %q: %w", group.Name, err)
		}
	}

	for _, balance := range backup.BalanceSnapshots {
		if balance.PulledAt.IsZero() {
			return fmt.Errorf("relay site balance backup requires pulled at")
		}
		create := client.RelaySiteBalanceSnapshot.Create().
			SetRelaySiteID(siteID).
			SetBalance(balance.Balance).
			SetPulledAt(balance.PulledAt)
		if balance.Unit != "" {
			create.SetUnit(balance.Unit)
		}
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site balance backup: %w", err)
		}
	}

	for _, modelPrice := range backup.ModelPrices {
		if modelPrice.ModelID == "" {
			return fmt.Errorf("relay site model price backup requires model id")
		}
		if modelPrice.SyncedAt.IsZero() {
			return fmt.Errorf("relay site model price backup %q requires synced at", modelPrice.ModelID)
		}
		if _, err := client.RelaySiteModelPrice.Create().
			SetRelaySiteID(siteID).
			SetModelID(modelPrice.ModelID).
			SetPrice(modelPrice.Price).
			SetSyncedAt(modelPrice.SyncedAt).
			Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site model price backup %q: %w", modelPrice.ModelID, err)
		}
	}

	for _, checkinLog := range backup.CheckinLogs {
		if checkinLog.ExecutedAt.IsZero() {
			return fmt.Errorf("relay site checkin log backup requires executed at")
		}
		create := client.RelaySiteCheckinLog.Create().
			SetRelaySiteID(siteID).
			SetExecutedAt(checkinLog.ExecutedAt).
			SetStatus(checkinLog.Status).
			SetNillableMessage(checkinLog.Message).
			SetNillableErrorMessage(checkinLog.ErrorMessage)
		if _, err := create.Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site checkin log backup: %w", err)
		}
	}

	for _, announcement := range backup.Announcements {
		if announcement.RemoteID == "" {
			return fmt.Errorf("relay site announcement backup requires remote id")
		}
		if strings.TrimSpace(announcement.Content) == "" {
			return fmt.Errorf("relay site announcement backup %q requires content", announcement.RemoteID)
		}
		if announcement.ContentHash == "" {
			return fmt.Errorf("relay site announcement backup %q requires content hash", announcement.RemoteID)
		}
		if announcement.FetchedAt.IsZero() {
			return fmt.Errorf("relay site announcement backup %q requires fetched at", announcement.RemoteID)
		}
		if _, err := client.RelaySiteAnnouncement.Create().
			SetRelaySiteID(siteID).
			SetRemoteID(announcement.RemoteID).
			SetNillableType(announcement.Type).
			SetContent(strings.TrimSpace(announcement.Content)).
			SetNillableExtra(announcement.Extra).
			SetContentHash(announcement.ContentHash).
			SetNillablePublishedAt(announcement.PublishedAt).
			SetFetchedAt(announcement.FetchedAt).
			SetNillableReadAt(announcement.ReadAt).
			SetMetadata(announcement.Metadata).
			Save(ctx); err != nil {
			return fmt.Errorf("failed to create relay site announcement backup %q: %w", announcement.RemoteID, err)
		}
	}

	return nil
}
