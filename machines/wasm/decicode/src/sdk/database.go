package sdk

import (
	"reflect"
)

type MyDb struct {
	BaseDB *Db
	Users  *EntityGroup[User]
	Points *EntityGroup[Point]
	Docs   *EntityGroup[Doc]
}

type User struct {
	Id       string
	Name     string
	AuthCode string
}

type Point struct {
	Id        string
	CreatorId string
}

type Doc struct {
	Id        string
	Title     string
	CreatorId string
	Path      string
}

func NewMyDb() *MyDb {
	db := &Db{
		EntityTypes: map[string]any{},
	}
	docsColl := NewEntityType(db, Doc{})
	db.EntityTypes[docsColl.Id] = docsColl
	pointsColl := NewEntityType(db, Point{})
	db.EntityTypes[pointsColl.Id] = pointsColl
	usersColl := NewEntityType(db, User{})
	db.EntityTypes[usersColl.Id] = usersColl
	return &MyDb{
		BaseDB: db,
		Docs:   docsColl.Store,
		Users:  usersColl.Store,
		Points: pointsColl.Store,
	}
}

func InstantiateEntityGroup(db *Db, id string, prefix string) any {
	if (id == reflect.TypeOf(User{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[User]), db)
	} else if (id == reflect.TypeOf(Doc{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[Doc]), db)
	} else if (id == reflect.TypeOf(Point{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[Point]), db)
	}
	return nil
}
