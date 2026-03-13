package qb

/*

 qb is a simple query-builder for `dbapi` whose purpose is to write automated CRUD queries
 (select, insert, update & delete) for simple, single-table database entities.

 The query builder relies on types & interfaces defined in this package, e.g Table, Entity, PrimaryKey etc
 and the struct tags defined for each field on the entity implementing class.

 Table represents the single table & it's generic over Entity.

 Entity is an interface implemented by any table/query result that represents multiple columns read or
 inserted in a database table.

 PrimaryKey represents the key for that Entity

*/
