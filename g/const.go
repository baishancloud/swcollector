package g

import (
	"time"
)

// changelog:
// 3.1.3: code refactor
// 3.1.4: bugfix ignore configuration
// 3.1.4: add localhost wan and lan traffic collect
// 3.1.5: add tag to set the collecter for switch data
// 3.1.6: bugfix
// 4.3.1: 修改交换机采集为多进程按端口方式采集，添加交换机端口丢包数据，添加本机网卡，iptables统计内外网流量数据，修改启动方式支持superior。
// 4.3.5: 修改交换机采集多进程出现锁问题。
// 4.3.8: 删除外调snmpwalk方式。
// 4.5.0: 重构代码使结构清晰一点
// 4.5.22: 添加raven 异常信息收集，pfc统计
// 4.5.23: 修复本地交换机扫描错误将linux服务器判断为交换机
// 4.5.25: 修复网口只配置IP时，列举对应网段IP列表出错
// 5.0.1: 采集的累进值数据可以换算成rate。
// 5.0.2: 传输目标配置改为数组。
// 5.0.3: 修改交换机的metric格式错误
// 5.0.5: 修改交换机采集者个数为1.
// 5.0.6: fix rpc timeout error 没有处理
// 5.0.7: fix trafffic metric 没有 endpoint 字段
// 5.0.8: fix 交换机不响应 ping 情况
// 5.1.2: 修改snmp采集方式为snmp getbluk
// 5.1.3: 修改主备探测方式为smudge
// 5.1.4: 改正snmp任务错误判断，调整主备日志记录。
// 5.1.7: 更新支持区分ipv6 流量采集.
// 5.1.8: 对collector ip 的smudge dead 状态增加curl check

const (
	VERSION          = "5.1.8"
	COLLECT_INTERVAL = time.Second
)
