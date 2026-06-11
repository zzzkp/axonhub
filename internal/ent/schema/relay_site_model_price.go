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

type RelaySiteModelPrice struct {
	ent.Schema
}

func (RelaySiteModelPrice) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}, schematype.SoftDeleteMixin{}}
}

func (RelaySiteModelPrice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "model_id", "deleted_at").
			StorageKey("relay_site_model_prices_by_site_model").
			Unique(),
	}
}

func (RelaySiteModelPrice) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.String("model_id").Immutable(),
		field.JSON("price", objects.RelaySiteRemoteModelPrice{}).Default(objects.RelaySiteRemoteModelPrice{}).Optional().
			Annotations(entgql.Skip(entgql.SkipAll)),
		field.Time("synced_at"),
	}
}

func (RelaySiteModelPrice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("model_prices").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteModelPrice) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteModelPrice) Policy() ent.Policy {
	return relaySitePolicy()
}
