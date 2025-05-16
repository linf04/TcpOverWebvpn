### 1、原理

![image](https://github.com/user-attachments/assets/c0b83de9-e3ce-4b85-b899-742677effa11)


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
![image](https://github.com/user-attachments/assets/cd95ee33-70f1-4a07-8c78-3c6f438ec451)


```shell
webvpn2tcp.exe -mode server -listen :12333 -target 127.0.0.1:81
```

```shell
webvpn2tcp.exe -mode client -server 10.69.12.112:12333 -listen :9999 -cookie "show_vpn=0; heartbeat=1; show_faq=0; wrdvpn_upstream_ip=10.*.*.3; wengine_vpn_ticketvpncas_ahut_edu_cn=2ab19******9dd; route=8768cab********9ad807f94da0a; route_widget=56b770208*******************a2bc26; refresh=0"
```

### 4、最终效果

iperf测速

![image](https://github.com/user-attachments/assets/ef8c9612-48dc-4103-856a-c9d067024969)


### 5、应用场景

p2p、nat打洞

### 6、备注

目前只支持ahut
