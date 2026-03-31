# 本地 SQLite 爬虫

一个简单的 Go 命令行爬虫：

- 从指定起始 URL 开始抓取
- 只抓取同一域名下的 HTML 页面
- 提取页面标题、正文文本、摘要
- 使用 `SQLite` 把数据保存在本地文件
- 自带 `list` 和 `show` 命令查看抓取结果

## 构建命令行工具

```bash
go build -o crawler .
```

生成后可直接执行：

```bash
./crawler help
```

## 抓取页面

```bash
./crawler crawl \
  -url https://example.com \
  -db data/crawler.db \
  -max-pages 20 \
  -delay 500ms \
  -timeout 10s
```

兼容旧用法：

```bash
go run . -url https://example.com
```

## 查看列表

查看最近 10 条：

```bash
./crawler list -db data/crawler.db -limit 10
```

按域名过滤：

```bash
./crawler list -db data/crawler.db -host example.com -limit 10
```

## 查看详情

按 ID 查看：

```bash
./crawler show -db data/crawler.db -id 1
```

按 URL 查看：

```bash
./crawler show -db data/crawler.db -url https://example.com/about
```

## 命令参数

`crawl`:

- `-url`: 起始页面，必填
- `-db`: SQLite 文件路径，默认 `data/crawler.db`
- `-max-pages`: 最多抓取多少个页面
- `-delay`: 每次请求之间的等待时间
- `-timeout`: 单个请求超时时间
- `-user-agent`: 自定义请求头

`list`:

- `-db`: SQLite 文件路径
- `-limit`: 显示多少条记录
- `-host`: 只看指定域名

`show`:

- `-db`: SQLite 文件路径
- `-id`: 按记录 ID 查看
- `-url`: 按页面 URL 查看

## 数据表

程序会自动创建 `pages` 表，主要字段如下：

- `url`: 页面地址，唯一
- `host`: 域名
- `title`: 标题
- `summary`: 摘要
- `body_text`: 提取后的正文文本
- `status_code`: HTTP 状态码
- `content_type`: 响应类型
- `depth`: 抓取深度
- `crawled_at`: 抓取时间

## 如果你想直接查 SQLite

```bash
sqlite3 data/crawler.db "select id, url, title, crawled_at from pages order by crawled_at desc limit 10;"
```
