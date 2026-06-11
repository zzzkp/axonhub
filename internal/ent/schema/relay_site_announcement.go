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

type RelaySiteAnnouncement struct {
	ent.Schema
}

func (RelaySiteAnnouncement) Mixin() []ent.Mixin {
	return []ent.Mixin{TimeMixin{}}
}

func (RelaySiteAnnouncement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relay_site_id", "remote_id").
			StorageKey("relay_site_announcements_by_site_remote").
			Unique(),
		index.Fields("relay_site_id", "read_at"),
		index.Fields("relay_site_id", "published_at"),
	}
}

func (RelaySiteAnnouncement) Fields() []ent.Field {
	return []ent.Field{
		field.Int("relay_site_id").Immutable(),
		field.String("remote_id"),
		field.String("type").Optional().Nillable(),
		field.Text("content"),
		field.Text("extra").Optional().Nillable(),
		field.String("content_hash"),
		field.Time("published_at").Optional().Nillable(),
		field.Time("fetched_at"),
		field.Time("read_at").Optional().Nillable(),
		field.JSON("metadata", objects.RelaySiteAnnouncementMetadata{}).Default(objects.RelaySiteAnnouncementMetadata{}).Optional().
			Annotations(entgql.Skip(entgql.SkipAll)),
	}
}

func (RelaySiteAnnouncement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("relay_site", RelaySite.Type).
			Ref("announcements").
			Field("relay_site_id").
			Required().
			Immutable().
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (RelaySiteAnnouncement) Annotations() []schema.Annotation {
	return []schema.Annotation{entgql.RelayConnection()}
}

func (RelaySiteAnnouncement) Policy() ent.Policy {
	return relaySitePolicy()
}
