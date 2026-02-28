# 骊珠修行笔记

本目录用于存放修行笔记，通过 `lizhu note add` 入库后，护道人对话时会自动检索相关内容。

## 目录结构

```
notes/
├── go/        Go 开发相关笔记（语言特性、标准库、并发、工程实践等）
├── ai/        AI 应用相关笔记（模型调用、Agent 设计、Prompt 工程等）
├── daily/     日常修行日志（YYYY-MM-DD.md 格式，记录每日学习收获与心得）
├── code/      代码片段与实战总结（具体问题的解决方案、踩坑记录等）
└── README.md  本说明文件
```

## 使用方式

```bash
# 单个文件入库
lizhu note add ./notes/go/concurrency.md

# 查看已入库文件
lizhu note list
```

## 写作建议

- 每篇笔记聚焦一个主题，篇幅适中（300-2000字为佳）
- Markdown 格式，段落之间空一行（系统按段落分块向量化）
- 修改笔记后重新 `lizhu note add` 即可更新，无需手动删除旧记录
