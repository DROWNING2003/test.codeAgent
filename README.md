# 本地 SQLite 爬虫

一个用 Go 编写的轻量级命令行爬虫，适合把网站页面抓到本地并保存到 `SQLite`，后续再做搜索、分析或导出。

## 功能

- 从指定起始 URL 开始抓取页面
- 只跟进同一域名下的链接
- 只保存 HTML 页面
- 提取页面标题、摘要和纯文本正文
- 使用本地 `SQLite` 持久化数据
- 自带 `crawl`、`list`、`show` 命令查看结果

## 适用场景

- 快速采集某个网站的公开页面
- 本地保存页面文本，方便后续检索或分析
- 做一个小型站点采集工具，而不是依赖外部数据库

## 快速开始

要求：

- Go 1.25+
- 无需本地 C 编译环境
  当前使用 `modernc.org/sqlite`（纯 Go 实现），默认构建不依赖 CGO 工具链

构建命令行工具：

```bash
go build -o crawler .
```

查看帮助：

```bash
./crawler help
```

## 常用流程

1. 抓取一个站点

```bash
./crawler crawl \
  -url https://example.com \
  -db data/crawler.db \
  -max-pages 20 \
  -delay 500ms \
  -timeout 10s
```

2. 查看最近抓到的页面

```bash
./crawler list -db data/crawler.db -limit 10
```

3. 查看某条记录详情

```bash
./crawler show -db data/crawler.db -id 1
```

## 命令说明

### `crawl`

抓取页面并写入本地数据库。

```bash
./crawler crawl \
  -url https://example.com \
  -db data/crawler.db \
  -max-pages 50 \
  -delay 300ms \
  -timeout 15s \
  -user-agent "local-sqlite-crawler/1.0"
```

参数：

- `-url`: 起始页面，必填
- `-db`: SQLite 文件路径，默认 `data/crawler.db`
- `-max-pages`: 最多抓取多少个页面
- `-delay`: 请求之间的等待时间
- `-timeout`: 单个请求超时时间
- `-user-agent`: 自定义请求头

兼容旧用法：

```bash
go run . -url https://example.com
```

### `list`

查看数据库里最近保存的页面。

```bash
./crawler list -db data/crawler.db -limit 10
./crawler list -db data/crawler.db -host example.com -limit 10
```

参数：

- `-db`: SQLite 文件路径
- `-limit`: 显示多少条记录
- `-host`: 只查看指定域名

输出字段：

- `ID`: 记录 ID
- `HOST`: 域名
- `STATUS`: HTTP 状态码
- `DEPTH`: 抓取深度
- `CRAWLED_AT`: 抓取时间
- `TITLE`: 页面标题
- `URL`: 页面地址

### `show`

查看某条记录的完整信息，包括正文文本。

```bash
./crawler show -db data/crawler.db -id 1
./crawler show -db data/crawler.db -url https://example.com/about
```

参数：

- `-db`: SQLite 文件路径
- `-id`: 按记录 ID 查看
- `-url`: 按页面 URL 查看

## 数据存储

程序会自动创建 `pages` 表，主要字段如下：

- `id`: 自增主键
- `url`: 页面地址，唯一
- `host`: 域名
- `title`: 标题
- `summary`: 摘要
- `body_text`: 提取后的正文文本
- `status_code`: HTTP 状态码
- `content_type`: 响应类型
- `depth`: 抓取深度
- `crawled_at`: 抓取时间

默认数据库路径：

```text
data/crawler.db
```

## 直接查看 SQLite

如果你本机装了 `sqlite3`，也可以直接查：

```bash
sqlite3 data/crawler.db "select id, url, title, crawled_at from pages order by crawled_at desc limit 10;"
```

## 注意事项

- 当前只抓取同域名链接，不会跨站点扩散
- 当前只处理静态 HTML，不执行 JavaScript
- 页面内容提取是轻量实现，适合通用文本采集，不是精细化正文抽取器
- 如果目标站点有反爬限制、频率限制或访问规则，需要自行调整 `-delay`、`-timeout` 和 `-user-agent`
