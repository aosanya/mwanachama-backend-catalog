// Package routes builds the HTTP surface over a
// [mwanachamacatalog.CatalogManager].
//
// It decides nothing about who may call what. The storefront read, the
// operator's read of the same Entry, the private parts and the whole mining
// backlog are separate addresses precisely so the mounting process can put
// a different gate on each — the anonymous reader gets OpenEntry and a page
// view, and nothing else. Mounting EntryRoutes without that gate publishes
// every pending agency.
package routes
