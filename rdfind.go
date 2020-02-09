package main

import (
    "os"
    "fmt"
    "path/filepath"
    "crypto/sha512"
    "io"
    "io/ioutil"
    "encoding/hex"
    "sync"
    "strconv"
    "strings"
)

const (
    separator = string(os.PathSeparator)
    version = "v2.0"
)

type FileInfos struct {
    size int64
    fpath string
}

var filestorage map[string][]FileInfos = make(map[string][]FileInfos)
var mutex sync.Mutex

func showhelp () {
    fmt.Println("program version: ", version)
    fmt.Println("-d folder")
    fmt.Println("-w handle way:")
    fmt.Println("    0.show same file (default)")
    fmt.Println("    1.delete new find file")
    fmt.Println("    2.change new find file to a hard link")
    fmt.Println("    3.change new find file to a symlink, need super permission")
    fmt.Println("-e delete empty folder")
    fmt.Println("-j delete size is 0 file")
    fmt.Println("-v or --version show program version")
    fmt.Println("-h or --help show help")
}

func gethash(fpath string) (int64, string, error) {
    file, err := os.Open(fpath)
    if err != nil {
        return 0, "", err
    }
    defer file.Close()
    h_ob := sha512.New()
    size, err2 := io.Copy(h_ob, file)
    if err2 != nil {
        return 0, "", err2
    }
    hash := h_ob.Sum(nil)
    hashvalue := hex.EncodeToString(hash)
    return size, hashvalue, nil
}

func removeemptyfolder (fpath string, wg *sync.WaitGroup) {
    dir, err := ioutil.ReadDir(fpath)
    if err != nil {
        fmt.Println(err)
        return
    }
    if len(dir) == 0 {
        err2 := os.Remove(fpath)
        if err2 != nil {
            fmt.Println(err2)
        }
        index := strings.LastIndex(fpath, separator)
        path := fpath[:index]
        removeemptyfolder(path, nil)
    }
    if wg != nil {
        wg.Done()
    }
}

func run (size int64, fpath string, way int, removeempty bool, delzerofile bool, wg *sync.WaitGroup) {
    defer wg.Done()
    fmt.Println("start check file:", fpath)
    if size == 0 && delzerofile == true {
        fmt.Println("file size is 0, delete", fpath)
        err := os.Remove(fpath)
        if err != nil {
            fmt.Println(err)
        }
        return
    }
    _, key, err := gethash(fpath)
    if err != nil {
        fmt.Println(err)
        return
    }
    mutex.Lock()
    defer mutex.Unlock()
    val, ok := filestorage[key]
    if ok == false { // key不存在，新建
        filestorage[key] = []FileInfos{{size, fpath}}
        return
    }
    // key存在，追加
    findsamefile := false
    vallen := len(val)
    for i :=0 ; i < vallen ; i++ {
        if val[i].size == size { // 文件大小相同
            fmt.Println("find a same file", val[i].fpath)
            findsamefile = true
        }
    }
    if findsamefile == false { // 未找到其他文件大小与sha512均相同的文件
        filestorage[key] = append(val, FileInfos{size, fpath})
        return
    }
    // 找到其他文件大小与sha512均相同的文件
    if way == 0 { // 什么都不做
        filestorage[key] = append(val, FileInfos{size, fpath})
    } else if way == 1 { // 删除新发现的
        fmt.Println("delete file", fpath)
        err2 := os.Remove(fpath)
        if err2 != nil {
            fmt.Println(err2)
        }
        if removeempty {
            index := strings.LastIndex(fpath, separator)
            path := fpath[:index]
            removeemptyfolder(path, nil)
        }
    } else if way == 2 { // 删除新发现的，然后创建硬链接
        filestorage[key] = append(val, FileInfos{size, fpath})
        fmt.Println("delete file and create a hard link", fpath)
        err2 := os.Remove(fpath)
        if err2 != nil {
            fmt.Println(err2)
        }
        err3 := os.Link(val[0].fpath, fpath)
        if err3 != nil {
            fmt.Println(err3)
        }
    } else if way == 3 { // 删除新发现的，然后创建软链接
        filestorage[key] = append(val, FileInfos{size, fpath})
        fmt.Println("delete file and create a symlink", fpath)
        err2 := os.Remove(fpath)
        if err2 != nil {
            fmt.Println(err2)
        }
        err3 := os.Symlink(val[0].fpath, fpath)
        if err3 != nil {
            fmt.Println(err3)
        }
    }
}

func main () {
    folder := ""
    way := 0
    removeempty := false
    delzerofile := false
    args := os.Args
    if args == nil {
        showhelp()
        return
    }
    argslen := len(args)
    for i := 1 ; i < argslen ; i++ {
        if args[i] == "-d" {
            i++
            folder = args[i]
        } else if args[i] == "-w" {
            i++
            val, err := strconv.Atoi(args[i])
            if err != nil {
                showhelp()
                return
            }
            way = val
        } else if args[i] == "-e" {
            removeempty = true
        } else if args[i] == "-j" {
            delzerofile = true
        } else if args[i] == "-v" || args[i] == "--version" {
            fmt.Println(version)
            return
        } else if args[i] == "-h" || args[i] == "--help" {
            showhelp()
            return
        }
    }
    if folder == "" {
        showhelp()
        return
    }
    wg := sync.WaitGroup{}
    filepath.Walk(folder, func (path string, info os.FileInfo, err error) error {
        if err != nil {
            fmt.Println(err)
            return nil
        }
        if info.IsDir() == false { // 只有是普通文件才计算与判断
            wg.Add(1)
            go run(info.Size(), path, way, removeempty, delzerofile, &wg)
        } else if removeempty {
            wg.Add(1)
            go removeemptyfolder(path, &wg)
        }
        return nil
    })
    wg.Wait()
}
