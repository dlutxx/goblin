package main

import (
    "io/ioutil"
    "log"
    "flag"
    "fmt"
    "strings"
    "time"
    "net/http"
    g "github.com/dlutxx/goblin"
)

var addr = flag.String("addr", ":8888", "server address")

const (
    MaxPostSize = 10 * 1 << 20
)

type HttpBin struct {
    scheme string
}

func (hb *HttpBin) Home(ctx *g.Context) {
    ctx.Res.SetHeader("Content-Type", "text/html; charset=utf-8")
    ctx.Res.WriteString(`This is a demo project to mimic the famous <a href="http://httpbin.org">httpbin.org</a>`)
}

func (hb *HttpBin) returnJSON(res *g.Response, v interface{}) {
    res.SetHeader("Content-Type", "application/json; charset=utf-8")
    res.WriteJSON(v)
}

func (hb *HttpBin) getIP(req *g.Request) string {
    parts := strings.Split(req.RemoteAddr, ":")
    return strings.Join(parts[:len(parts)-1], ":")
}

func (hb *HttpBin) IP(ctx *g.Context) {
    data := map[string]string {
        "origin": hb.getIP(ctx.Req),
    }
    hb.returnJSON(ctx.Res, data)
}

func (hb *HttpBin) getHeaders(req *g.Request) map[string]string {
    headers := map[string]string{}
    for k, vals := range req.Header {
        headers[k] = strings.Join(vals, ", ")
    }
    return headers
}

func (hb *HttpBin) Headers(ctx *g.Context) {
    hb.returnJSON(ctx.Res, hb.getHeaders(ctx.Req))
}

func (hb *HttpBin) UserAgent(ctx *g.Context) {
    data := map[string]string {
        "user-agent": ctx.Req.UserAgent(),
    }
    hb.returnJSON(ctx.Res, data)
}

func (hb *HttpBin) getArgs(req *g.Request) map[string]interface{} {
    query := req.URL.Query()
    args := map[string]interface{}{}
    for k, vals := range query {
        if len(vals) > 1 {
            args[k] = vals
        } else {
            args[k] = vals[0]
        }
    }
    return args
}

func (hb *HttpBin) fullURL(req *g.Request) string {
    return fmt.Sprintf("%v://%v%v", hb.scheme, req.Host, req.RequestURI)
}

func (hb *HttpBin) Get(ctx *g.Context) {
    if ctx.Req.Method != "GET" {
        ctx.Res.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    data := map[string]interface{} {
        "url": hb.fullURL(ctx.Req),
        "args": hb.getArgs(ctx.Req),
        "origin": hb.getIP(ctx.Req),
        "headers": hb.getHeaders(ctx.Req),
    }
    hb.returnJSON(ctx.Res, data)
}

func (hb *HttpBin) Post(ctx *g.Context) {
    if ctx.Req.Method != "POST" {
        ctx.Res.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    fileData := map[string]interface{}{}
    formData := map[string]interface{}{}
    if err := ctx.Req.ParseMultipartForm(MaxPostSize); err != nil {
        panic("invalid multipart form data")
    }
    for k, vals := range ctx.Req.MultipartForm.Value {
        if len(vals) > 1 {
            formData[k] = vals
        } else {
            formData[k] = vals[0]
        }
    }
    for k, vals := range ctx.Req.MultipartForm.File {
        contents := []string{}
        for _, fh := range vals {
            file, err := fh.Open()
            if err != nil {
                contents = append(contents, err.Error())
                continue
            }
            content, _ := ioutil.ReadAll(file)
            contents = append(contents, string(content))
        }
        if len(contents) > 1 {
            fileData[k] = contents
        } else {
            fileData[k] = contents[0]
        }
    }
    data := map[string]interface{} {
        "url": hb.fullURL(ctx.Req),
        "args": hb.getArgs(ctx.Req),
        "origin": hb.getIP(ctx.Req),
        "headers": hb.getHeaders(ctx.Req),
        "files": fileData,
        "form": formData,
    }
    hb.returnJSON(ctx.Res, data)
}

func (hb *HttpBin) Status(ctx *g.Context) {
    code, _ := ctx.Params.Int("code")
    ctx.Res.WriteHeader(code)
}

func (hb *HttpBin) Redirect(ctx *g.Context) {
    num, _ := ctx.Params.Int("num")
    var next string
    if num > 1 {
        next = fmt.Sprintf("/redirect/%v", num - 1)
    } else {
        next = "/get"
    }
    ctx.Res.RedirectTemp(next)
}

func (hb *HttpBin) Delay(ctx *g.Context) {
    num, _ := ctx.Params.Int("num")
    time.Sleep(time.Duration(num) * time.Second)
    hb.Get(ctx)
}

func main() {
    flag.Parse()

    hb := &HttpBin{"http"}
    cfg := g.Settings{
        g.CfgKeyRoutes: map[string]interface{}{
            `^$`: g.HF(hb.Home),
            `^ip$`: g.HF(hb.IP),
            `^headers$`: g.HF(hb.Headers),
            `^user-agent$`: g.HF(hb.UserAgent),
            `^get$`: g.HF(hb.Get),
            `^post$`: g.HF(hb.Post),
            `^status/(?P<code>\d{3})$`: g.HF(hb.Status),
            `^redirect/(?P<num>\d+)$`: g.HF(hb.Redirect),
            `^delay/(?P<num>\d{1,2})$`: g.HF(hb.Delay),
        },
    }
    app := g.NewApp(cfg)
    log.Fatalln(app.ListenAndServe(*addr))
}