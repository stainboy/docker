package file

import (
    "os"
    "bufio"
    "io"
    "time"
    SAPredis "github.wdf.sap.corp/sigma-anywhere/saptail/redis"
    "github.com/garyburd/redigo/redis"

    "fmt"
    "syscall"
    "bytes"
    "compress/zlib"
    "encoding/gob"
    "errors"
)

type MapStr map[string]interface{}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

type Stat_t syscall.Stat_t
type Redis SAPredis.Config
type Reader struct {
    file   string
    offset int64
}
type FStat struct {
    filename string
    stat     Stat_t
    offset   int64
}
type Tail struct {
    Redis RedisInterface
}
type Interface interface {
    GetFileStat(filename string) (*syscall.Stat_t, error)
    SetRedis(redisURL, redisPort string, DB int) (error)
}
type RedisInterface interface {
    SetRedis(redisURL, redisPort string, DB int) (error)
    GetOffset(filename string, params ...int) (int64, error)
    SetOffset(filename string, Offset int64, params ...int) (error)
    GetFileStat(filename string, params ...int) (*syscall.Stat_t, error)
    SetFileStat(filename string, stat *syscall.Stat_t, params ...int) (error)
}

func New() (*Tail) {
    return &Tail{Redis: new(Redis)}
}
func (t *Tail) SetRedis(redisURL, redisPort string, DB int) (error) {
    return t.Redis.SetRedis(redisURL, redisPort, DB)
}
func (f *Reader) Read(p []byte) (n int, err error) {
    reader, err := os.Open(f.file)
    defer reader.Close()
    if err != nil {
        return 0, err
    }
    reader.Seek(f.offset, 0)

    n, err = reader.Read(p)

    if err == io.EOF {
        time.Sleep(1 * time.Second)
    }
    f.offset += int64(n)
    return n, err
}

func (t *Tail) GetFileStat(filename string) (*syscall.Stat_t, error) {
    finfo, err := os.Stat(filename)
    return finfo.Sys().(*syscall.Stat_t), err
}

func (t *Tail) MonitorFile(filename string, zip bool) {
    go t.monitorFile(filename, zip)
}
func (t *Tail) monitorFile(filename string, zip bool) {
    var err error
    offset, err := t.Redis.GetOffset(filename)
    checkErr(err)
    var lines string
    reader := &Reader{filename, offset}
    br := bufio.NewReader(reader)
    for {
        log, _, err := br.ReadLine()
        if err == io.EOF {
            break
        }

        if err != nil {
            continue
        }
        if len(lines) == 0 {
            lines = string(log)
        } else {
            lines = lines + "\n" + string(log)
        }
    }
    var message interface{}

    if zip {
        message = DoZlibCompress([]byte(lines))
    } else {
        message = lines
    }
    e := MapStr{
        "@timestamp": time.Now().UTC(),
        "source":     reader.file,
        "message":    message,
        "offset":     offset,
        "zip":        zip,
    }
    t.Redis.SetOffset(filename, reader.offset)
    fmt.Println(e)
}

func DoZlibCompress(src []byte) []byte {
    var in bytes.Buffer
    w := zlib.NewWriter(&in)
    w.Write(src)
    w.Close()
    return in.Bytes()
}
func DoZlibUnCompress(compressSrc []byte) []byte {
    b := bytes.NewReader(compressSrc)
    var out bytes.Buffer
    r, _ := zlib.NewReader(b)
    io.Copy(&out, r)
    return out.Bytes()
}

func (r *Redis) SetRedis(redisURL, redisPort string, DB int) (error) {
    r.Port = redisPort
    r.Url = redisURL
    r.DB = DB
    r.Pool = SAPredis.NewPool(redisURL+":"+redisPort, DB)
    return nil
}

func (r *Redis) GetOffset(filename string, params ...int) (int64, error) {
    var DB int
    if len(params) == 0 {
        DB = 2
    } else {
        DB = params[0]
    }
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)

    reply, err := client.Do("GET", filename)
    checkErr(err)
    if reply == nil {
        return 0, errors.New("file does not exist")
    } else {
        re, err := redis.Int64(reply, nil)
        return re, err
    }
}
func (r *Redis) SetOffset(filename string, Offset int64, params ...int) (error) {
    var DB int
    if len(params) == 0 {
        DB = 2
    } else {
        DB = params[0]
    }
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)
    r.readmeOffset(DB)

    _, err := client.Do("SET", filename, Offset)
    return err

}
func (r *Redis) readmeOffset(DB int) {
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)
    reply, err := redis.String(client.Do("GET", "README"))
    readme := "This db is for Offset, you can use `GET <KEY>`"
    if err != nil {
        client.Do("SET", "README", readme)
    } else if reply != readme {
        client.Do("SET", "README", readme)
    }
}

func (r *Redis) GetFileStat(filename string, params ...int) (*syscall.Stat_t, error) {
    var DB int
    var err error
    if len(params) == 0 {
        DB = 3
    } else {
        DB = params[0]
    }
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)
    //c.Client.Do("SELECT", DB)

    //reply, err := redis.Values(client.Do("HGETALL", filename))
    var stat syscall.Stat_t
    //err = redis.ScanStruct(reply, &stat)
    //checkErr(err)

    b, err := redis.Bytes(client.Do("GET", filename))
    if err != nil {
        checkErr(err)
    }
    //var s syscall.Stat_t
    var network bytes.Buffer
    network.Write(b)
    dec := gob.NewDecoder(&network)
    dec.Decode(&stat)
    //fmt.Println(s)

    return &stat, err
}
func (r *Redis) SetFileStat(filename string, stat *syscall.Stat_t, params ...int) (error) {
    var DB int
    var err error
    if len(params) == 0 {
        DB = 3
    } else {
        DB = params[0]
    }
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)
    r.readmeFileStat(DB)

    var network bytes.Buffer
    enc := gob.NewEncoder(&network)
    err = enc.Encode(stat)
    if err != nil {
        checkErr(err)
    }
    //fmt.Println(network.Bytes())
    client.Do("SET", filename, network.Bytes())
    //_, err = client.Do("HMSET", redis.Args{filename}.AddFlat(stat)...)
    return err
}
func (r *Redis) readmeFileStat(DB int) {
    client := r.Pool.Get()
    defer client.Close()
    client.Do("SELECT", DB)
    //readme := "This db is for FileStat, you can use `HGETALL <KEY>`"
    readme := "This db is for FileStat"
    reply, err := redis.String(client.Do("GET", "README"))
    if err != nil {
        client.Do("SET", "README", readme)
    } else if reply != readme {
        client.Do("SET", "README", readme)
    }
}
