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

type RelaySiteGroup struct {
	ent.Schema
}

func (RelaySiteGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, schematype.SoftDeleteMixin{}}
}

func (RelaySiteGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "name", "deleted_at").
			StorageKey("relay_site_groups_by_site_name").
			Unique(),
	}
}

func (RelaySiteGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.String("name").Immutable(),
		field.Float("ratio").Optional().Nillable(),
		field.JSON("settings", objects.RelaySiteGroupSettings{}).Default(objects.RelaySiteGroupSettings{}).Optional().
			Annotations(entgql.Skip(entgql.SkipAll)),
		field.Time("synced_at"),
	}
}

func (RelaySiteGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("groups").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteGroup) Policy() ent.Policy {
	return relaySitePolicy()
}
