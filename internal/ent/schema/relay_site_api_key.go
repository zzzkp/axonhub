package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/objects"
)

type RelaySiteAPIKey struct {
	ent.Schema
}

func (RelaySiteAPIKey) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, schematype.SoftDeleteMixin{}}
}

func (RelaySiteAPIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "remote_id", "deleted_at").
			StorageKey("relay_site_api_keys_by_site_remote_id").
			Unique(),
		index.Fields("relay_site_id", "status"),
		index.Fields("relay_site_id", "group_name"),
	}
}

func (RelaySiteAPIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.String("remote_id").Immutable(),
		field.String("name").Optional(),
		field.Enum("status").Values("enabled", "disabled", "unknown").Default("unknown"),
		field.String("group_name").Optional().Nillable(),
		field.Float("quota").Optional().Nillable(),
		field.Float("used_quota").Optional().Nillable(),
		field.Time("expires_at").Optional().Nillable(),
		field.JSON("metadata", objects.RelaySiteAPIKeyMetadata{}).Default(objects.RelaySiteAPIKeyMetadata{}).Optional().
			Annotations(entgql.Skip(entgql.SkipAll)),
		field.Time("synced_at"),
	}
}

func (RelaySiteAPIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("api_keys").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteAPIKey) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteAPIKey) Policy() ent.Policy {
	return relaySitePolicy()
}
