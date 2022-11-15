# gox

[![Go Reference](https://pkg.go.dev/badge/github.com/wencan/gox)](https://pkg.go.dev/github.com/wencan/gox)  


Go语言（非官方）基础库。  
lockfree数据结构，已经迁往：[freesync](https://github.com/wencan/freesync)。

## 目录
<table>
    <tr>
        <th>包</th><th>结构体或方法</th><th>作用</th><th>说明</th>
    </tr>
    <tr>
        <td rowspan="1">xcontainer</td><td>ListMap</td><td>同时具备List和Map的特性的容器</td><td></td>
    </tr>
    <tr>
        <td rowspan="2">xsync<br><i>（已经迁移到<a href="https://github.com/wencan/freesync">freesync</a>)</i></td><td><a href="https://pkg.go.dev/github.com/wencan/freesync#Slice">Slice</a></td><td>并发安全的Slice结构</td><td>与官方slice+mutex相比，写性能提升一半，读性能提升百倍左右</td>
    </tr>
    <tr>
        <td><a href="https://pkg.go.dev/github.com/wencan/freesync#Bag">Bag</a></td><td>并发安全的容器</td><td>与sync.Map相比，写性能提升一半左右</td>
    </tr>
    <tr>
        <td rowspan="2">xsync</td><td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync#LRUMap">LRUMap</a></td><td>并发安全的LRU结构</td><td>与GroupCache的LRU相比，写性能相当，读性能提升近百倍</td>
    </tr>
    <tr>
        <td>xsync/sentinel</td><td><a href="https://pkg.go.dev/github.com/wencan/gox/xsync/sentinel#SentinelGroup">SentinelGroup</a></td><td>哨兵机制</td><td>同singleflight，但支持批量处理</td>
    </tr>
    <tr>
        <td><a href="https://pkg.go.dev/github.com/wencan/gox/async">async</a></td><td><a href="https://pkg.go.dev/github.com/wencan/gox/async#Series">Series</a><br><a href="https://pkg.go.dev/github.com/wencan/gox/async#Parallel">Parallel</a></td><td></td><td>协程和异步任务的辅助方法</td>
    </tr>
</table>