package domain

import (
	goose "jaytaylor.com/GoOse"
	archiveis "jaytaylor.com/archive.is"
	hn "jaytaylor.com/hn-utils/domain"
)

// Article wraps goose.Article with extra attributes specific to the circus show.
// Comprises a "hydrated" article.
type Article struct {
	*goose.Article

	NamedEntities NamedEntities `json:"namedEntities"`
}

// Context holds an entire story context, including metadata.
type Context struct {
	*hn.Story
	Article   *Article             `json:"Goose"`
	ArchiveIs []archiveis.Snapshot `json:"Archiveis"`
}
