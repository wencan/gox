# gox

[![Go Reference](https://pkg.go.dev/badge/github.com/wencan/gox)](https://pkg.go.dev/github.com/wencan/gox)  


Go语言（非官方）基础库。

## 目录
<table>
    <tr>
        <th>包</th><th>结构体</th><th>作用</th><th>说明</th>
    </tr>
    <tr>
        <td rowspan="1">xcontainer</td><td>ListMap</td><td>支持LRU缓存</td><td>现在LRU缓存更好的选择是xsync/LRUMap</td>
    </tr>
    <tr>
        <td rowspan="2">xsync</td><td>Slice</td><td>并发安全的Slice结构</td><td>与官方slice+mutex相比，写性能相当，读性能提升不少</td>
    </tr>
    <tr>
        <td>LRUMap</td><td>并发安全的LRU结构</td><td>与GroupCache的LRU相比，写性能相当，读性能提升很多</td>
    </tr>
    <tr>
        <td>xsync/sentinel</td><td>SentinelGroup</td><td>哨兵机制</td><td>同singleflight，但支持批量处理</td>
    </tr>
</table>