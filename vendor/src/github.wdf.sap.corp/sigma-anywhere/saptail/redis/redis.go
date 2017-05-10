package redis

import (
    "github.com/garyburd/redigo/redis"
    "github.com/docker/docker/daemon/logger"
    "log"
    "time"
    "bytes"
    "os"
    "strconv"
)

//var pool *redis.Pool
//var idleTime = 10

type Config struct {
    Url  string
    Port string
    DB   int
    Pool *redis.Pool
    //Client redis.Conn
    flash bool // TODO: this flash should be chan bool, need to be improved
}

type Interface interface {
    SendMessage(msg *logger.Message, lazymode bool) (error)
    //HealthCheck() (string, error)
    Close() (error)
    flush() (error)
    sendMessage(msg *logger.Message) (error)
    //sendOut(key string) (error)
    //writeFile(filename string, line []byte) (error)
}
//type Stat_t syscall.Stat_t

func NewPool(addr string, DB int) *redis.Pool {
    return &redis.Pool{
        MaxIdle:     3,
        IdleTimeout: 240 * time.Second,
        Dial: func() (redis.Conn, error) {
            c, err := redis.Dial("tcp", addr)
            if err != nil {
                return nil, err
            }
            c.Do("SELECT", DB)
            return c, nil
        },
    }
}
func checkErr(err error, msg ...interface{}) {
    if err != nil {
        log.Printf("%v", err)
        log.Printf("%v", msg)
    }
}
func New(redisURL, redisPort string, DB int) (Interface, error) {
    var err error
    pool := NewPool(redisURL+":"+redisPort, DB)
    err = os.MkdirAll("/var/log/container/", 0755)
    checkErr(err)

    var R Interface
    R = &Config{
        //Client: client,
        Url:   redisURL,
        Port:  redisPort,
        Pool:  pool,
        DB:    DB,
        flash: false,
    }
    go func() {
        for {
            R.flush()
        }
    }()
    //go func() {
    //    for {
    //        _, err := R.HealthCheck()
    //        checkErr(err)
    //        time.Sleep(time.Second)
    //    }
    //}()
    return R, nil
}

//func (c *Config) HealthCheck() (string, error) {
//    client := c.Pool.Get()
//    res, err := client.Do("ping")
//    defer client.Close()
//    s, err := redis.String(res, err)
//    return s, err
//}
func (c *Config) flush() (error) {
    //if c.flash {
        time.Sleep(time.Second)
    //}
    client := c.Pool.Get()
    err := client.Flush()
    defer client.Close()
    return err
}

func (c *Config) Close() (error) {
    c.flush()
    return c.Pool.Close()
}

func (c *Config) SendMessage(msg *logger.Message, lazymode bool) (error) {
    var err error
    if lazymode {

    } else {
        //client := c.Pool.Get()
        //err = client.Send("LPUSH", msg.Attrs["logkey"], msg.Line)
        //defer client.Close()
        if len(msg.Attrs["logkey"]) > 0 {
            err = c.sendMessage(msg)
            checkErr(err, msg)
        }
        //c.SetOffset(msg.Source, msg.Offset)
        //c.SetFileStat(msg.Source, msg.Stat_t)
    }
    return err
}
func (c *Config) sendMessage(msg *logger.Message) (error) {
    var err error
    buf := new(bytes.Buffer)
    client := c.Pool.Get()
    buf.WriteString(strconv.FormatInt(msg.Timestamp.UnixNano(), 10))
    buf.WriteString("\t")
    buf.Write(msg.Line)
    err = client.Send("LPUSH", msg.Attrs["logkey"], buf.Bytes())
    defer client.Close()
    checkErr(err, msg)
    return err
}

//func (c *Redis) sendOut(key string) (error) {
//    var err error
//    if (!Tofile[key]) {
//        // TODO: quota check and string check
//        client := c.Pool.Get()
//        err = client.Send("LPUSH", key, Buf[key])
//        defer client.Close()
//        checkErr(err, key, Buf[key])
//
//    } else {
//        err = c.writeFile("/var/log/container/"+key+".log", Buf[key])
//        checkErr(err)
//    }
//    return err
//
//}
//
//func (c *Redis) writeFile(filename string, line []byte) (error) {
//    var err error
//    if (len(line) == 0) {
//        return nil
//    }
//    f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
//    checkErr(err, "os.OpenFile")
//    _, err = f.Write(line)
//    checkErr(err, "f.Write", line, string(line))
//    _, err = f.WriteString("\n")
//    defer f.Close()
//    return err
//}
