package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/objects"
)

type RelaySiteCredential struct {
	ent.Schema
}

func (RelaySiteCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (RelaySiteCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id").Unique(),
	}
}

func (RelaySiteCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.Enum("auth_type").Values("token", "password"),
		field.JSON("credential", objects.RelaySiteCredential{}).Sensitive(),
	}
}

func (RelaySiteCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("credential").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.Skip(entgql.SkipAll),
	}
}

func (RelaySiteCredential) Policy() ent.Policy {
	return relaySitePolicy()
}
