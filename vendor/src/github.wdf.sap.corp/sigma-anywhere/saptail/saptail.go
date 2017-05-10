package saptail

import (
    "log"
    "github.wdf.sap.corp/sigma-anywhere/saptail/redis"
    "github.com/docker/docker/daemon/logger"
)

type Taylor struct {
    Redis redis.Interface
}
type Interface interface {
    Message(msg *logger.Message, lazymode bool) (error)
    Close() (error)
}

func checkErr(err error) {
    if err != nil {
        log.Printf("%v", err)
    }
}

func New(URL, Port string, DB int) (Interface, error) {
    rd, err := redis.New(URL, Port, DB)
    return &Taylor{Redis: rd}, err
}

func (t *Taylor) Message(msg *logger.Message, lazyMode bool) (error) {
    err := t.Redis.SendMessage(msg, lazyMode)
    checkErr(err)
    return err
}
func (t *Taylor) Close() (error) {
    return t.Redis.Close()
}
