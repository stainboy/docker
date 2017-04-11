package file

import (
    "os"
    "bufio"
    "io"
    "time"
    "github.wdf.sap.corp/sigma-anywhere/saptail/redis"
    "fmt"
    "syscall"
    "bytes"
    "compress/zlib"
)

type MapStr map[string]interface{}

func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

type Stat_t syscall.Stat_t

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
    Client redis.Config
}
type Interface interface {
    GetFileStat(filename string) (*syscall.Stat_t, error)
}

func New() (Interface, error) {
    var F Interface
    F = &Tail{
    }
    return F, nil
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
    offset, err := t.Client.GetOffset(filename)
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
    t.Client.SetOffset(filename, reader.offset)
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
