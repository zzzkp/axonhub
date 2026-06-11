package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RelaySiteBalanceSnapshot struct {
	ent.Schema
}

func (RelaySiteBalanceSnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (RelaySiteBalanceSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "pulled_at").
			StorageKey("relay_site_balance_snapshots_by_site_pulled_at"),
	}
}

func (RelaySiteBalanceSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.Float("balance"),
		field.String("unit").Default("USD"),
		field.Time("pulled_at"),
	}
}

func (RelaySiteBalanceSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("balance_snapshots").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteBalanceSnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteBalanceSnapshot) Policy() ent.Policy {
	return relaySitePolicy()
}
