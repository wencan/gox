# gox

[![Go Reference](https://pkg.go.dev/badge/github.com/wencan/gox)](https://pkg.go.dev/github.com/wencan/gox)  


Go语言（非官方）基础库。

## 目录
<table>
    <tr>
        <th>包</th><th>结构体或方法</th><th>作用</th><th>说明</th>
    </tr>
    <tr>
        <td rowspan="1">xcontainer</td><td>ListMap</td><td>支持LRU缓存</td><td>现在LRU缓存更好的选择是xsync/LRUMap</td>
    </tr>
    <tr>
        <td rowspan="3">xsync</td><td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync#Slice">Slice</a></td><td>并发安全的Slice结构</td><td>与官方slice+mutex相比，读写性能提升不少</td>
    </tr>
    <tr>
        <td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync#Bag">Bag</a></td><td>并发安全的对写多友好的容器</td><td>支持添加、删除、不保证顺序的遍历。与sync.Map相比，写性能提升一半左右</td>
    </tr>
    <tr>
        <td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync#LRUMap">LRUMap</a></td><td>并发安全的LRU结构</td><td>与GroupCache的LRU相比，写性能相当，读性能提升很多</td>
    </tr>
    <tr>
        <td>xsync/sentinel</td><td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync/sentinel#SentinelGroup">SentinelGroup</a></td><td>哨兵机制</td><td>同singleflight，但支持批量处理</td>
    </tr>
    <tr>
        <td><a href="https://pkg.go.dev/github.com/wencan/gox/async">async</a></td><td><a href="https://pkg.go.dev/github.com/wencan/gox/async#Series">Series</a><br><a href="https://pkg.go.dev/github.com/wencan/gox/async#Parallel">Parallel</a></td><td></td><td>协程和异步任务的辅助方法</td>
    </tr>
</table>