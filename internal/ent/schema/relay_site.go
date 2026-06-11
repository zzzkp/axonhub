package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/scopes"
)

type RelaySite struct {
	ent.Schema
}

func (RelaySite) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		schematype.SoftDeleteMixin{},
	}
}

func (RelaySite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "deleted_at").
			StorageKey("relay_sites_by_name").
			Unique(),
		index.Fields("type"),
		index.Fields("status"),
	}
}

func (RelaySite) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Annotations(entgql.OrderField("NAME")),
		field.Enum("type").
			Values("new_api").
			Default("new_api").
			Comment("Relay site type. new_api represents a new-api relay site.").
			Annotations(entgql.OrderField("TYPE")),
		field.String("base_url"),
		field.Enum("status").
			Values("enabled", "disabled", "archived").
			Default("disabled").
			Annotations(entgql.OrderField("STATUS")),
		field.Bool("auto_checkin_enabled").
			Default(false).
			Annotations(entgql.OrderField("AUTO_CHECKIN_ENABLED")),
		field.String("remark").Optional().Nillable(),
		field.Time("last_synced_at").Optional().Nillable().
			Annotations(entgql.OrderField("LAST_SYNCED_AT"), entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
		field.String("last_sync_error").Optional().Nillable().
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
		field.Time("last_checkin_at").Optional().Nillable().
			Annotations(entgql.OrderField("LAST_CHECKIN_AT"), entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
		field.String("last_checkin_result").Optional().Nillable().
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
	}
}

func (RelaySite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("credential", RelaySiteCredential.Type).
			Unique().
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput)),
		edge.To("api_keys", RelaySiteAPIKey.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("groups", RelaySiteGroup.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("balance_snapshots", RelaySiteBalanceSnapshot.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("model_prices", RelaySiteModelPrice.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("checkin_logs", RelaySiteCheckinLog.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
		edge.To("announcements", RelaySiteAnnouncement.Type).
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput), entgql.RelayConnection()),
	}
}

func (RelaySite) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.RelayConnection(),
		entgql.Mutations(entgql.MutationCreate(), entgql.MutationUpdate()),
	}
}

func (RelaySite) Policy() ent.Policy {
	return relaySitePolicy()
}

func relaySitePolicy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.OwnerRule(),
			scopes.UserReadScopeRule(scopes.ScopeReadChannels),
		},
		Mutation: scopes.MutationPolicy{
			scopes.OwnerRule(),
			scopes.UserWriteScopeRule(scopes.ScopeWriteChannels),
		},
	}
}
