# Libre Hardware Monitor 集成

本文档介绍如何在Windows系统上使用Libre Hardware Monitor的web接口获取硬件监控数据。

## 概述

AX206Monitor现在支持通过Libre Hardware Monitor的HTTP API获取Windows系统的硬件监控数据，包括：

- CPU使用率、温度、频率
- GPU使用率、温度、频率  
- 内存使用情况
- 风扇转速
- 网络上传/下载速度

## 前置条件

### 1. 安装Libre Hardware Monitor

1. 从 [Libre Hardware Monitor官网](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor) 下载最新版本
2. 以管理员权限运行Libre Hardware Monitor
3. 在设置中启用HTTP服务器功能

### 2. 配置HTTP服务器

1. 打开Libre Hardware Monitor
2. 进入 `Options` -> `Web Server`
3. 勾选 `Run web server`
4. 设置端口（默认8085）
5. 确保防火墙允许该端口的连接

## 配置AX206Monitor

### 1. 配置文件设置

在配置文件中添加`libre_hardware_monitor_url`字段：

```json
{
  "name": "windows",
  "width": 480,
  "height": 320,
  "libre_hardware_monitor_url": "http://127.0.0.1:8085",
  "refresh_interval": 1000,
  "items": [
    {
      "type": "value",
      "monitor": "cpu_usage",
      "label": "CPU"
    },
    {
      "type": "value", 
      "monitor": "cpu_temp",
      "label": "CPU Temp"
    }
  ]
}
```

### 2. URL配置说明

- **默认URL**: `http://127.0.0.1:8085`
- **本地运行**: `http://localhost:8085` 或 `http://127.0.0.1:8085`
- **远程访问**: `http://[IP地址]:8085`

### 3. 支持的监控项

| 监控项 | 描述 | 单位 |
|--------|------|------|
| `cpu_usage` | CPU使用率 | % |
| `cpu_temp` | CPU温度 | °C |
| `cpu_freq` | CPU频率 | MHz |
| `gpu_usage` | GPU使用率 | % |
| `gpu_temp` | GPU温度 | °C |
| `gpu_freq` | GPU频率 | MHz |
| `memory_usage` | 内存使用率 | % |
| `memory_used` | 已用内存 | GB |
| `memory_total` | 总内存 | GB |
| `cpu_fan_speed` | CPU风扇转速 | RPM |
| `gpu_fan_speed` | GPU风扇转速 | RPM |
| `net_upload` | 网络上传速度 | MB/s |
| `net_download` | 网络下载速度 | MB/s |

## 使用示例

### 1. 使用预配置的Windows配置

```bash
ax206monitor -config windows
```

### 2. 自定义配置

创建自己的配置文件，参考`config/windows.json`：

```json
{
  "libre_hardware_monitor_url": "http://localhost:8085",
  "items": [
    {
      "type": "big_value",
      "x": 10,
      "y": 40,
      "width": 200,
      "height": 60,
      "monitor": "cpu_temp",
      "label": "CPU温度"
    },
    {
      "type": "chart",
      "x": 10,
      "y": 120,
      "width": 460,
      "height": 100,
      "monitor": "cpu_usage",
      "label": "CPU使用率历史"
    }
  ]
}
```

## 故障排除

### 1. 连接问题

如果无法连接到Libre Hardware Monitor：

1. 确认Libre Hardware Monitor正在运行
2. 检查HTTP服务器是否已启用
3. 验证URL和端口是否正确
4. 检查防火墙设置

### 2. 数据获取问题

如果某些监控项显示为0或不可用：

1. 确认硬件支持相应的传感器
2. 在Libre Hardware Monitor中检查传感器是否可见
3. 检查传感器名称是否匹配解析逻辑

### 3. 性能优化

- 数据缓存1秒，避免频繁请求
- HTTP超时设置为5秒
- 如果Libre Hardware Monitor不可用，自动回退到WMI

## 日志调试

启用调试日志查看详细信息：

```bash
# 设置环境变量启用调试日志
export LOG_LEVEL=debug
ax206monitor -config windows
```

查看日志中的相关信息：
- `Fetching data from Libre Hardware Monitor`
- `Successfully fetched and parsed data`
- `Failed to fetch data from Libre Hardware Monitor`

## 网络配置

### 局域网访问

如果Libre Hardware Monitor运行在其他机器上：

1. 确保目标机器的防火墙允许8085端口
2. 在配置文件中使用正确的IP地址
3. 测试网络连通性：`telnet [IP] 8085`

### 安全考虑

- Libre Hardware Monitor的HTTP接口没有身份验证
- 建议仅在受信任的网络环境中使用
- 可以通过防火墙限制访问来源

## 示例数据格式

Libre Hardware Monitor返回的JSON数据结构示例：

```json
{
  "id": 0,
  "Text": "Sensor",
  "Children": [
    {
      "Text": "12th Gen Intel Core i5-12490F",
      "Children": [
        {
          "Text": "Load",
          "Children": [
            {
              "Text": "CPU Total",
              "Value": "19.4 %",
              "Type": "Load",
              "SensorId": "/intelcpu/0/load/0"
            }
          ]
        }
      ]
    }
  ]
}
```

## 更多信息

- [Libre Hardware Monitor GitHub](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor)
- [AX206Monitor项目主页](https://github.com/yukunyi/ax206monitor)
