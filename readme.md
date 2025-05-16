### 1、原理

![](C:\Users\admin\AppData\Roaming\marktext\images\2025-05-16-17-44-09-image.png)

使用webvpn的websocket转发流量。

### 2、作用

在校内设备和校外设备建立高速转发端口。

### 3、用法

校内服务器端：

```shell
webvpn2tcp.exe -mode server -listen <校内服务端监听地址> -target <转发校内ip:port>
```

校外客户端：

```shell
webvpn2tcp.exe -mode client -server <校内服务端ip> -listen  <:映射端口> -cookie <webvpn Cookie>
```

示例：

登录webvpn门户，复制cookie

![](C:\Users\admin\AppData\Roaming\marktext\images\2025-05-16-17-55-30-image.png)

```shell
webvpn2tcp.exe -mode server -listen :12333 -target 127.0.0.1:81
```

```shell
webvpn2tcp.exe -mode client -server 10.69.12.112:12333 -listen :9999 -cookie "show_vpn=0; heartbeat=1; show_faq=0; wrdvpn_upstream_ip=10.*.*.3; wengine_vpn_ticketvpncas_ahut_edu_cn=2ab19******9dd; route=8768cab********9ad807f94da0a; route_widget=56b770208*******************a2bc26; refresh=0"
```

### 4、最终效果

iperf测速

![](C:\Users\admin\AppData\Roaming\marktext\images\2025-05-16-18-00-41-image.png)

### 5、应用场景

p2p、nat打洞

### 6、备注

目前只支持ahut
