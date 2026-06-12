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

type RelaySiteCheckinLog struct {
	ent.Schema
}

func (RelaySiteCheckinLog) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (RelaySiteCheckinLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "executed_at").
			StorageKey("relay_site_checkin_logs_by_site_executed_at"),
		index.Fields("relay_site_id", "status"),
	}
}

func (RelaySiteCheckinLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.Time("executed_at"),
		field.Enum("status").Values("success", "failed", "skipped"),
		field.String("message").Optional().Nillable(),
		field.String("error_message").Optional().Nillable(),
	}
}

func (RelaySiteCheckinLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("checkin_logs").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteCheckinLog) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteCheckinLog) Policy() ent.Policy {
	return relaySitePolicy()
}
