// Package mwanachamacatalog is a catalog of documents with a review state:
// things are proposed, built, reviewed, published or shared under a key, and
// counted. It exposes [CatalogManager].
//
// # It names no domain, and that is the point
//
// Nothing in this package knows what a document is about. It was written for
// an agency storefront and is expected to run a library, a course catalog
// and a product catalog without a line changing, so every type here is named
// for its shape rather than its subject.
//
// The rule is one sentence: **a word that means something in one domain and
// nothing in another does not belong in this package.** "Sector" fails it —
// a library has genres, not sectors. "Entry" passes it.
//
// Concretely, three fields already failed that test and were moved out:
//
//   - Entry.Sector and Entry.Archetype were columns. They are agency words,
//     so they now live inside Doc and are filtered by JSON path — see
//     ListFilter.Docs and [models.DocFilter].
//   - Suggestion.Sector was a column. Gone the same way, into Proposal.
//   - PageKind's values were "home" and "agency". They are now "index" and
//     "entry": a storefront's front page and a library's front page are the
//     same shape, and the page of one thing is the page of one thing.
//
// What survives beside Doc is only what means the same everywhere: a Slug, a
// Name, a Summary, a review state, timestamps. Everything else a domain
// cares about goes in the document, where the domain's own spec says which
// paths are worth filtering on.
//
// A worked comparison — one store, two domains:
//
//	Entry + Doc      a mined agency archetype  |  a title's record
//	EntryPrivate     readme, sales, configs    |  acquisition notes, supplier terms
//	ShareLink        preview an unpublished    |  preview an uncatalogued title
//	                 agency                    |
//	Suggestion       an archetype worth mining |  a patron purchase request
//	SuggestionMatch  the agency that answered  |  the copy acquired
//	MiningRun        a mine-agencies run       |  a MARC import run
//	PageView         views per agency          |  views per title
//
// When adding anything here, the test to apply is whether a librarian would
// recognise it. If the answer needs the word "agency" to make sense, it
// belongs in the document or in the caller, not in this package.
//
// # Why it is its own module
//
// Neither of the two modules that looked closest could hold it.
// mwanachama-backend-forms models the lifecycle better than the catalog's
// original SQL did — its PublicLink keeps key, status and retired_at where
// the SQL overwrote a single hash, and its Approval is a real
// submit/withdraw/approve/refuse — but a Form's content is its Questions,
// and there is no JSON, blob or payload field in any of its models.
// mwanachama-backend-assetmanager maps a slug onto Code and a grouping onto
// Category cleanly, then has nowhere for a 34 KB document but
// AttributesJSON, a plain text column its own comment calls a stand-in, and
// carries Movement and Hold that a catalog never uses. ShareLink and the
// review state are modelled on forms's shapes rather than reinvented.
//
// # Storage
//
// GORM directly, in the shape mwanachama-backend-assetmanager and
// mwanachama-backend-actor settled on: models/ holds the domain types,
// gormstore/ the row structs and migration, one file per entity in each.
// Entry.Doc is jsonb wherever the database has it — see gormstore.JSONDoc —
// because filtering and counting inside the document happen in the database.
//
// Layout:
//   - models/    — domain types (Entry, EntryPrivate, ShareLink, Suggestion,
//     MiningRun, Setting, PageView) and DocFilter
//   - gormstore/ — GORM row structs, row<->domain conversion, migration
//   - doc.go (this file), tables.go — table-name/migrate wrappers
//   - catalog_manager.go — CatalogManager interface, catalogManager struct
//   - entry.go           — Entry CRUD, visibility and the public read
//   - private.go         — the operator's notes and module configs
//   - sharelink.go       — issuing, retiring and checking share keys
//   - suggestion.go      — the backlog, its attempts and its stall
//   - run.go             — run reports, settings and page views
//   - errors.go          — sentinel errors
package mwanachamacatalog
