插件化架构：
输入插件：支持文件尾监听 (TailInput) 和 TCP 端口监听 (TcpInput)
过滤插件：实现了基于正则的日志过滤 (GrepFilter) 和字段转换 (RecordTransformerFilter)
输出插件：支持标准输出 (StdoutOutput) 和文件输出 (FileOutput)
事件处理流程：
事件 (Event) 封装了日志的标签、时间戳和内容
使用队列 (Queue) 在不同组件间传递事件，实现解耦
采用多 goroutine 并发处理，提高效率
关键特性：
支持文件读取位置记录 (pos_file)，避免重复读取
缓冲区机制，支持批量处理和定时刷新
标签匹配系统，实现事件的定向处理
优雅的启动和关闭机制，确保资源正确释放

启动
```
go run main.go
```