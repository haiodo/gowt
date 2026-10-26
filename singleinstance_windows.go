package gowt

func singleInstance(id string, args []string) (bool, error) { return singleInstanceSocket(id, args) }
