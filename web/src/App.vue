<script setup>
import {computed, onMounted, provide, ref} from "vue";
import moment from 'moment'
import CPU from "@/components/CPU.vue";
import Mem from "@/components/Mem.vue";
import NetIn from "@/components/NetIn.vue";
import NetOut from "@/components/NetOut.vue";
import axios from "axios";
import { Message } from "@arco-design/web-vue";
import StatsCard from "@/components/StatsCard.vue";
import {formatAgo, formatBytes, formatTimeStamp, formatUptime, formatUptimeZh, calculateRemainingDays} from '@/utils/utils'
import HeaderLocale from "@/components/HeaderLocale.vue";
import {useI18n} from "vue-i18n";

const { t } = useI18n()

const socketURL = ref('')
const apiURL = ref('')
const siteName = ref('Monitor Party')
const offlineWait = ref(60)

const theme = window.localStorage.getItem('theme') || 'light'
const dark = ref(theme !== 'light')

const handleChangeDark = () => {
  dark.value = !dark.value

  if (dark.value) {
    window.localStorage.setItem('theme','dark')
    document.body.setAttribute('arco-theme', 'dark')
  } else {
    // 恢复亮色主题
    window.localStorage.setItem('theme','light')
    document.body.removeAttribute('arco-theme');
   }
}

const area = ref([])
const selectArea = ref('all')

const type = ref('all')

const data = ref([])

const selectHost = ref('')

const charts = ref({})

const cpuRef = ref(null)
const memRef = ref(null)
const netInRef = ref(null)
const netOutRef = ref(null)

const host = computed(() => {
  if (selectArea.value === 'all') {
    return data.value
  }

  return data.value.filter(item => item.Host.Name.slice(0, 2) === selectArea.value)
})

const hosts = computed(() => {
  if (type.value === 'all') {
    return host.value
  } else if (type.value === 'online') {
    return host.value.filter(item => item.status)
  } else {
    return host.value.filter(item => !item.status)
  }
})

const stats = computed(() => {
  const online = host.value.filter(item => item.status)
  let bandwidth_up = 0
  let bandwidth_down = 0
  let traffic_up = 0
  let traffic_down = 0

  host.value.forEach((item) => {
    bandwidth_up += item.State.NetOutSpeed
    bandwidth_down += item.State.NetInSpeed
    traffic_up += item.State.NetOutTransfer
    traffic_down += item.State.NetInTransfer
  })

  return {
    total: host.value.length,
    online: online.length,
    offline: host.value.length - online.length,
    bandwidth_up: bandwidth_up,
    bandwidth_down: bandwidth_down,
    traffic_up: traffic_up,
    traffic_down: traffic_down
  }
})

let socket = null

let nowtime = (Math.floor(Date.now() / 1000))

const fetchConfig = async () => {
  try {
    const res = await axios.get('/config.json')
    socketURL.value = res.data.socket
    apiURL.value = res.data.apiURL
    siteName.value = res.data.siteName || 'Monitor Party'
    offlineWait.value = Number(res.data.offlineWait) || 60
    document.title = siteName.value
  } catch (e) {
    Message.error(t('get-config-error'))
  }
}

const initScoket = async () => {
  await fetchConfig()

  socket = new WebSocket(socketURL.value);  // 替换为实际的 WebSocket 服务器 URL

  socket.onmessage = function(event) {
    try {
      const message = event.data;
      const res = JSON.parse(message.replace('data: ', ''))
      if (res != null) {
        area.value = Array.from(new Set(res.map(item => item.Host.Name.slice(0, 2))))
      }
      data.value = res.map((host) => {
        if (!charts.value[host.Host.Name]) {
          charts.value[host.Host.Name] = {
            cpu: [],
            mem: [],
            net_in: [],
            net_out: []
          }
        }

        if (charts.value[host.Host.Name].cpu.length == 2) {
          charts.value[host.Host.Name].cpu = charts.value[host.Host.Name].cpu.slice(1)
          charts.value[host.Host.Name].mem = charts.value[host.Host.Name].mem.slice(1)
          charts.value[host.Host.Name].net_in = charts.value[host.Host.Name].net_in.slice(1)
          charts.value[host.Host.Name].net_out = charts.value[host.Host.Name].net_out.slice(1)
        }

        charts.value[host.Host.Name].cpu.push([host.TimeStamp * 1000, host.State.CPU])
        charts.value[host.Host.Name].mem.push([host.TimeStamp * 1000, host.State.MemUsed])
        charts.value[host.Host.Name].net_in.push([host.TimeStamp * 1000, host.State.NetOutSpeed])
        charts.value[host.Host.Name].net_out.push([host.TimeStamp * 1000, host.State.NetInSpeed])

        return {
          ...host,
          status: (host.TimeStamp > 0 && nowtime - host.TimeStamp <= offlineWait.value) ? 1 : 0
        }
      })

      setTimeout(() => sendPing(), 1000)

    } catch (error) {
      console.error(t('ws-error'), error);
    }
  };

  socket.onopen = function () {
    sendPing()
  }

  socket.onclose = function () {
    Message.warning(t('ws-error-reconnect'))

    initScoket()
  }
}

const sendPing = () => {
  nowtime = (Math.floor(Date.now() / 1000))
  socket.send('ping')
}

onMounted(async() => {
  if (dark.value) {
    document.body.setAttribute('arco-theme', 'dark')
  }

  await initScoket()
  handleFetchHostInfo()
})

const progressStatus = (value) => {
  if (value < 80) {
    return 'success';
  } else if (value < 90) {
    return 'warning';
  } else {
    return 'danger';
  }
}

const handleSelectArea = (area) => {
  selectArea.value = area
}

const handleSelectHost = (host) => {
  handleFetchHostInfo()
  if (selectHost.value === host) {
    selectHost.value = ''
    return
  }

  selectHost.value = host
}

const hostInfo = ref({})

const handleFetchHostInfo = async () => {
  try {
    const res = await axios.get(apiURL.value + '/info')
    hostInfo.value = {}
    res.data.forEach((item) => {
      hostInfo.value[item.name] = item
    })
  } catch (e) {
    // Message.error('删除失败，管理密钥错误')
  }
}

const handleChangeType = (value) => {
  type.value = value
}

const getHostInfo = (name) => {
  return hostInfo.value[name] || hostInfo.value[name.trim()] || {}
}

const diskPercent = (item) => {
  if (!item.State.DiskTotal) return 0
  return item.State.DiskUsed / item.State.DiskTotal * 100
}

const memoryPercent = (item) => {
  if (!item.Host.MemTotal) return 0
  return item.State.MemUsed / item.Host.MemTotal * 100
}

const cpuCoresText = (item) => {
  const physical = item.Host.PhysicalCores || item.Host.LogicalCores || item.Host.CPU?.length || 0
  const logical = item.Host.LogicalCores || item.Host.CPU?.length || physical
  return `${physical} 物理 / ${logical} 逻辑`
}

const normalizeDueTime = (value) => {
  if (!value) return 0
  return Number(value) > 0 && Number(value) < 1000000000000 ? Number(value) * 1000 : value
}

provide('handleChangeType', handleChangeType)

</script>

<template>
  <div class="max-container">
    <div class="header">
      <a class="logo" href="#">
        <span class="brand-mark">M</span>
        <span>{{siteName}}</span>
        <small>实时节点观测台</small>
      </a>
      <a-space>
        <a-button class="admin-link" href="/admin" tag="a">管理后台</a-button>
        <HeaderLocale />
        <a-button class="theme-btn" :shape="'round'" @click="handleChangeDark">
          <template #icon>
            <icon-sun-fill v-if="!dark" />
            <icon-moon-fill v-else />
          </template>
        </a-button>
      </a-space>
    </div>
    <div class="area-tabs">
      <div class="area-tab-item" :class="selectArea === 'all' ? 'is-active' : ''" @click="handleSelectArea('all')">
        {{$t('all-area')}}
      </div>
      <div class="area-tab-item" :class="selectArea === item ? 'is-active' : ''" v-for="(item, index) in area" :key="item" @click="handleSelectArea(item)">
        <span :class="`flag-icon flag-icon-${item.replace('UK', 'GB').toLowerCase()}`" style="margin-right: 3px;"></span> {{item}}
      </div>
    </div>
    <StatsCard :type="type" :stats="stats" @handleChangeType="handleChangeType" />
    <div class="monitor-card">
      <div class="monitor-item" :class="selectHost === item.Host.Name ? 'is-active' : ''" v-for="(item, index) in hosts" @click="handleSelectHost(item.Host.Name)" :key="index">
        <div class="name">
          <div class="title">
            <span :class="`flag-icon flag-icon-${item.Host.Name.slice(0, 2).replace('UK', 'GB').toLowerCase()}`"></span>
            {{item.Host.Name}}
          </div>
          <div class="status" :class="item.status ? 'online' : 'offline'">
            <span>{{item.status  ? $t('online') : $t('offline')}}</span>
            <span style="margin-left: 6px;">{{formatUptime(item.State.Uptime)}}</span>
          </div>
        </div>
        <div class="platform">
          <div class="monitor-item-title">{{ $t('system') }}</div>
          <div class="monitor-item-value">{{item.Host.Platform}}</div>
        </div>
        <div class="cpu">
          <div class="monitor-item-title">CPU</div>
          <div class="monitor-item-value">{{item.State.CPU.toFixed(2) + '%'}}</div>
          <a-progress class="monitor-item-progress" :status="progressStatus(item.State.CPU)" :percent="item.State.CPU/100" :show-text="false" style="width: 60px" />
        </div>
        <div class="mem">
          <div class="monitor-item-title">{{ $t('memory') }}</div>
          <div class="monitor-item-value">{{memoryPercent(item).toFixed(2) + '%'}}</div>
          <a-progress class="monitor-item-progress" :status="progressStatus(memoryPercent(item))" :percent="memoryPercent(item)/100" :show-text="false" style="width: 60px" />
        </div>
        <div class="disk">
          <div class="monitor-item-title">硬盘</div>
          <div class="monitor-item-value">{{diskPercent(item).toFixed(2) + '%'}}</div>
          <a-progress class="monitor-item-progress" :status="progressStatus(diskPercent(item))" :percent="diskPercent(item)/100" :show-text="false" style="width: 60px" />
        </div>
        <div class="network">
          <div class="monitor-item-title">{{ $t('network') }} (IN|OUT)</div>
          <div class="monitor-item-value">{{`${formatBytes(item.State.NetInSpeed)}/s | ${formatBytes(item.State.NetOutSpeed)}/s`}}</div>
        </div>
        <div class="average">
          <div class="monitor-item-title">{{ $t('load') }} (1|5|15)</div>
          <div class="monitor-item-value">{{`${item.State.Load1} | ${item.State.Load5} | ${item.State.Load15}`}}</div>
        </div>
        <div class="uptime" style="width: 120px;">
          <div class="monitor-item-title">{{ $t('due-time-only') }}</div>
          <div class="monitor-item-value">{{hostInfo[item.Host.Name] ? calculateRemainingDays(hostInfo[item.Host.Name].due_time) : '-'}}</div>
        </div>
        <div class="detail" v-if="selectHost === item.Host.Name">
          <div class="purchase-info" v-if="getHostInfo(item.Host.Name).show_purchase_info">
            <div class="purchase-title">购买信息</div>
            <div class="purchase-grid">
              <div>
                <span>卖家</span>
                <strong>{{getHostInfo(item.Host.Name).seller || '-'}}</strong>
              </div>
              <div>
                <span>价格</span>
                <strong>{{getHostInfo(item.Host.Name).price || '-'}}</strong>
              </div>
              <div>
                <span>周期</span>
                <strong>{{getHostInfo(item.Host.Name).cycle || '-'}}</strong>
              </div>
              <div>
                <span>带宽</span>
                <strong>{{getHostInfo(item.Host.Name).bandwidth || '-'}}</strong>
              </div>
              <div>
                <span>月流量</span>
                <strong>{{getHostInfo(item.Host.Name).traffic || '-'}}</strong>
              </div>
              <div>
                <span>购买链接</span>
                <a v-if="getHostInfo(item.Host.Name).buy_url" :href="getHostInfo(item.Host.Name).buy_url" target="_blank" @click.stop="() => {}">{{getHostInfo(item.Host.Name).buy_url}}</a>
                <strong v-else>-</strong>
              </div>
            </div>
          </div>
          <a-row>
            <a-col :span="10" :xs="24" :sm="24" :md="10" :lg="10" :sl="10">
              <div class="detail-section-title">系统</div>
              <div class="detail-item-list">
                <div class="detail-item">
                  <div class="name">{{ $t('hostname') }}</div>
                  <div class="value">{{item.Host.Hostname || item.Host.Name}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('system') }}</div>
                  <div class="value">{{item.Host.Platform}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">内核</div>
                  <div class="value">{{item.Host.Kernel || item.Host.PlatformVersion || '-'}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('arch') }}</div>
                  <div class="value">{{item.Host.Arch || '-'}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('virtualization') }}</div>
                  <div class="value">{{item.Host.Virtualization || '-'}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">CPU 型号</div>
                  <div class="value">{{item.Host.CPUModel || '-'}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">核心</div>
                  <div class="value">{{cpuCoresText(item)}}</div>
                </div>
              </div>
              <div class="detail-section-title">网络与负载</div>
              <div class="detail-item-list">
                <div class="detail-item">
                  <div class="name">累计接收</div>
                  <div class="value">{{formatBytes(item.State.NetInTransfer)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">累计发送</div>
                  <div class="value">{{formatBytes(item.State.NetOutTransfer)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">磁盘读</div>
                  <div class="value">{{formatBytes(item.State.DiskReadSpeed)}}/s</div>
                </div>
                <div class="detail-item">
                  <div class="name">磁盘写</div>
                  <div class="value">{{formatBytes(item.State.DiskWriteSpeed)}}/s</div>
                </div>
                <div class="detail-item">
                  <div class="name">进程数</div>
                  <div class="value">{{item.State.Processes || 0}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">TCP / UDP</div>
                  <div class="value">{{item.State.TCP || 0}} / {{item.State.UDP || 0}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">运行时长</div>
                  <div class="value">{{formatUptimeZh(item.State.Uptime)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">数据更新</div>
                  <div class="value">{{formatAgo(item.TimeStamp)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">CPU{{ $t('use') }}</div>
                  <div class="value">{{item.State.CPU.toFixed(2) + '%'}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('memory') }}</div>
                  <div class="value">{{memoryPercent(item).toFixed(2) + '%'}} ({{formatBytes(item.State.MemUsed)}} / {{formatBytes(item.Host.MemTotal)}})</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('swap') }}</div>
                  <div class="value">{{formatBytes(item.State.SwapUsed)}} / {{formatBytes(item.Host.SwapTotal)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">硬盘总用量</div>
                  <div class="value">{{diskPercent(item).toFixed(2)}}% ({{formatBytes(item.State.DiskUsed)}} / {{formatBytes(item.State.DiskTotal)}})</div>
                </div>
                <div class="disk-list" v-if="item.State.Disks && item.State.Disks.length">
                  <div class="disk-title">硬盘明细</div>
                  <div class="disk-row" v-for="disk in item.State.Disks" :key="disk.mount">
                    <div class="disk-mount">{{disk.mount}} <small v-if="disk.fs_type">{{disk.fs_type}}</small></div>
                    <div class="disk-usage">{{disk.used_percent.toFixed(2)}}% · {{formatBytes(disk.used)}} / {{formatBytes(disk.total)}}</div>
                  </div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('network') }}（IN|OUT）</div>
                  <div class="value">{{`${formatBytes(item.State.NetInSpeed)}/s | ${formatBytes(item.State.NetOutSpeed)}/s`}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('load') }}(1|5|15)</div>
                  <div class="value">{{`${item.State.Load1} | ${item.State.Load5} | ${item.State.Load15}`}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('uptime') }}</div>
                  <div class="value">{{formatUptimeZh(item.State.Uptime)}}</div>
                </div>
                <div class="detail-item">
                  <div class="name">{{ $t('report-time') }}</div>
                  <div class="value">{{formatTimeStamp(item.TimeStamp)}}</div>
                </div>
                <div class="detail-item" v-if="hostInfo[item.Host.Name] && hostInfo[item.Host.Name].due_time">
                  <div class="name">{{ $t('due-time') }}</div>
                  <div class="value">{{moment(normalizeDueTime(hostInfo[item.Host.Name].due_time)).format('YYYY-MM-DD')}}</div>
                </div>
              </div>
            </a-col>
            <a-col :span="14" :xs="24" :sm="24" :md="14" :lg="14" :sl="14">
              <a-row :gutter="20">
                <a-col :span="12" :xs="24" :sm="24" :md="12" :lg="12" :sl="12">
                  <CPU ref="cpuRef" style="margin-bottom: 20px;" :data="charts[item.Host.Name].cpu" />
                </a-col>
                <a-col :span="12" :xs="24" :sm="24" :md="12" :lg="12" :sl="12">
                  <Mem ref="memRef" :max="item.Host.MemTotal" style="margin-bottom: 20px;" :data="charts[item.Host.Name].mem" />
                </a-col>
                <a-col :span="12" :xs="24" :sm="24" :md="12" :lg="12" :sl="12">
                  <NetIn ref="netInRef" :data="charts[item.Host.Name].net_in" />
                </a-col>
                <a-col :span="12" :xs="24" :sm="24" :md="12" :lg="12" :sl="12">
                  <NetOut ref="netOutRef" :data="charts[item.Host.Name].net_out" />
                </a-col>
              </a-row>
            </a-col>
          </a-row>
        </div>
      </div>
    </div>
    <div class="footer" style="margin-top: 30px">Monitor Party · Lightweight VPS telemetry</div>
    <div class="footer" style="margin-bottom: 30px">Copyright © {{new Date().getFullYear()}} Monitor Party.</div>
  </div>
</template>

<style lang="scss">
body {
  margin: 0;
  background: #f7f8fa;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Microsoft YaHei", sans-serif;
  color: #1f2937;
}

a {
  text-decoration: none;
}

.max-container {
  margin: 0 auto;
  width: 100%;
  max-width: 1500px;
}

.header {
  margin: 18px 14px 10px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 1px solid rgba(23, 33, 47, .08);
  border-radius: 14px;
  background: #fff;
  box-shadow: none;

  .theme-btn {
    border: 1px solid #d9e1ea!important;
    background-color: #ffffff!important;
    color: #17212f!important;
  }

  .admin-link {
    border-radius: 999px;
    background: #111827;
    color: #fff;
    border: none;
  }
}

.arco-dropdown {
  padding: 4px!important;
  border-radius: 8px!important;
  .arco-dropdown-option {
    border-radius: 4px !important;
    padding: 8px;
    line-height: 13px;
    font-size: 13px;
  }
}

.area-tabs {
  margin: 22px 14px 8px;

  .area-tab-item {
    margin-bottom: 10px;
    margin-right: 10px;
    padding: 8px 14px;
    border-radius: 999px;
    cursor: pointer;
    border: 1px solid rgba(31,41,55,.1);
    background: #fff;
    box-shadow: none;
    display: inline-block;

    .flag-icon {
      border-radius: 3px;
      margin-top: -3px;
    }

    &.is-active {
      background: #111827;
      color: #fff;
      border-color: #17212f;
    }
  }

}

.monitor-card {
  position: relative;
  margin: 0 auto;
  padding: 14px;

  .monitor-item {
    position: relative;
    margin-bottom: 14px;
    padding: 18px 26px;
    border-radius: 14px;
    border: 1px solid rgba(31,41,55,.08);
    display: block;
    background: #fff;
    box-shadow: none;
    cursor: pointer;

    &.is-active {
      background: #ffffff;
      border-color: rgba(17,24,39,.22);

      &>.detail {
        margin-top: 15px;
        border-top: 1px solid #eeeeee;
        padding-top: 15px;
        display: block;
      }
    }

    &:hover {
      background: #ffffff;

    }

    .flag-icon {
      margin-right: 5px;
      border-radius: 3px;
    }

    .monitor-item-title {
      margin-bottom: 3px;
      font-size: 11px;
      opacity: .6;
    }

    .monitor-item-value {
      font-weight: 500;
    }

    .monitor-item-progress {
      margin-top: 4px;
      height: 4px;
      display: block;
    }

    .detail {
      width: 100%;
    }

    .name {
      display: inline-block;
      vertical-align: middle;
      width: 250px;

      .title {
        margin-bottom: 5px;
        display: flex;
        align-items: center;
        font-weight: 600;
        font-size: 16px;
      }

      .status {
        display: flex;
        align-items: center;
        &::before {
          margin-right: 10px;
          position: relative;
          display: block;
          content: '';
          width: 8px;
          height: 8px;
          border-radius: 12px;
          background-color: #fb2c36;
        }

        &.online {
          &::before {
            background-color: #00c951;
          }
        }

        span {
          font-size: 13px;
          opacity: .6;
        }
      }
    }

    .platform {
      display: inline-block;
      vertical-align: top;
      width: 120px;
    }

    .cpu {
      display: inline-block;
      vertical-align: top;
      width: 120px;
    }

    .mem {
      display: inline-block;
      vertical-align: top;
      width: 120px;
    }

    .disk {
      display: inline-block;
      vertical-align: top;
      width: 120px;
    }

    .average {
      display: inline-block;
      vertical-align: top;
      width: 200px;
    }

    .network {
      display: inline-block;
      vertical-align: top;
      width: 200px;
    }

    .uptime {
      display: inline-block;
      vertical-align: middle;
      width: 200px;
    }

    .detail {
      display: none;

      .detail-item-list {
        margin-bottom: 20px;
      }

      .detail-section-title {
        margin: 12px 0 8px;
        font-size: 13px;
        font-weight: 800;
        color: #111827;
      }

      .purchase-info {
        margin: 0 0 16px;
        padding: 14px;
        border: 1px solid #e5e7eb;
        border-radius: 10px;
        background: #f9fafb;

        .purchase-title {
          margin-bottom: 8px;
          font-size: 13px;
          font-weight: 700;
          color: #111827;
        }

        .purchase-grid {
          display: grid;
          grid-template-columns: repeat(4, minmax(0, 1fr));
          gap: 10px;

          div {
            padding: 10px;
            border-radius: 8px;
            background: #fff;
            border: 1px solid #eef0f3;
            min-width: 0;
          }

          span {
            display: block;
            margin-bottom: 5px;
            font-size: 12px;
            color: #6b7280;
          }

          strong,
          a {
            font-size: 13px;
            color: #111827;
            font-weight: 600;
            word-break: break-all;
          }
        }
      }

      .disk-list {
        margin: 12px 0;
        padding: 12px;
        border: 1px solid #e5e7eb;
        border-radius: 10px;
        background: #fff;

        .disk-title {
          margin-bottom: 8px;
          font-size: 13px;
          font-weight: 700;
          color: #111827;
        }

        .disk-row {
          padding: 8px 0;
          border-top: 1px solid #f0f1f3;

          &:first-of-type {
            border-top: 0;
          }
        }

        .disk-mount {
          font-size: 12px;
          font-weight: 700;
          color: #111827;

          small {
            margin-left: 6px;
            color: #6b7280;
            font-weight: 500;
          }
        }

        .disk-usage {
          margin-top: 3px;
          font-size: 12px;
          color: #4b5563;
        }
      }

      .detail-item {
        .name {
          width: 30%;
          font-size: 12px;
          color: #666;
          margin-bottom: 5px;
          display: inline-block;
          vertical-align: top;
        }

        .value {
          width: 70%;
          font-size: 12px;
          font-weight: 500;
          display: inline-block;
          vertical-align: top;
        }
      }
    }
  }
}

.logo {
  margin-top: 0;
  margin-bottom: 0;
  position: relative;
  cursor: pointer;
  line-height: 42px;
  height: 42px;
  font-size: 16px;
  font-weight: 900;
  color: #17212f;
  display: flex;
  align-items: center;

  .brand-mark {
    margin-right: 10px;
    width: 38px;
    height: 38px;
    border-radius: 14px;
    display: grid;
    place-items: center;
    background: #111827;
    color: #fff;
    font-weight: 900;
  }

  small {
    margin-left: 10px;
    font-weight: 500;
    color: #6b7684;
  }
}

.monitor-action-bar {
  margin: 0 10px;
  display: inline-block;
  border: 1px solid #e5e5e5;
  background: #ffffff;
  box-shadow: 0 2px 4px 0 rgba(133, 138, 180, 0.14);
  border-radius: 4px;

  :deep(.arco-tabs-content) {
    display: none;
  }
}

.footer {
  line-height: 22px;
  text-align: center;
  color: #6b7684;

  a {
    color: rgba(var(--primary-6));
  }
}

body[arco-theme='dark'] {
  background-color: #111111;
  color: #ffffff;

  .arco-dropdown {
    background-color: #000000!important;
    border: 1px solid rgb(46 46 46)!important;
  }

  .arco-modal {
    background-color: #0e0e0e;
    border: 1px solid rgba(255,255,255,0.05);
  }

  .header {
    border-color: #333333;
    background: #000000;

    .logo {
      color: #ffffff;

      .brand-mark {
        background: #ffffff;
        color: #000000;
      }

      span,
      small {
        color: #ffffff;
      }
    }

    .admin-link {
      border: 1px solid #ffffff!important;
      background: #ffffff!important;
      color: #000000!important;
    }

    .theme-btn {
      border: 1px solid #333333!important;
      background-color: #000000!important;
      color: #ffffff!important;
    }
  }

  .area-tabs {
    .area-tab-item {
      border-color: #333333;
      background: #000000;
      color: #ffffff;
      box-shadow: none;

      &.is-active {
        background: #005fe705;
        color: #005fe7;
        border: 1px solid #005fe7;
      }
    }
  }

  .footer {
    color: #ffffffAA;
  }

  .monitor-card {
    .monitor-item {
      border: 1px solid rgb(255 255 255 / 16%);
      box-shadow: none;
      background-color: #000000;
      color: #ffffff;

      &:hover {
        background-color: #101010;
      }

      .detail {
        border-color: #333333AA;

        .purchase-info,
        .disk-list {
          border-color: #333333;
          background: #080808;

          .purchase-title,
          .disk-title,
          .disk-mount {
            color: #ffffff;
          }

          .disk-usage,
          small {
            color: #cccccc;
          }
        }

        .purchase-info .purchase-grid {
          div {
            border-color: #333333;
            background: #000000;
          }

          span {
            color: #aaaaaa;
          }

          strong,
          a {
            color: #ffffff;
          }
        }

        .detail-item-list {
          .detail-section-title {
            color: #ffffff;
          }

          .detail-item {
            .name {
              color: #999999;
            }

            .value {
              color: #ffffff;
            }
          }
        }
      }
    }
  }

}

@media screen and (max-width: 768px) {
  .monitor-card .monitor-item .detail .purchase-info .purchase-grid {
    grid-template-columns: 1fr;
  }

  .monitor-item {
    &>div {
      margin-bottom: 10px;
    }
  }

  .detail {
    .detail-item {
      .name {
        width: 25%!important;
      }
      .value {
        width: 75%!important;
      }
    }
  }
}

@media screen and (max-width: 1920px) {
  .max-container {
    max-width: 1440px;
  }
}

@media screen and (max-width: 1280px) {
  .max-container {
    max-width: 1140px;
  }
}

@media screen and (max-width: 992px) {
  .max-container {
    max-width: 960px;
  }
}

@media screen and (max-width: 768px) {
  .max-container {
    max-width: 720px;
  }
}

@media screen and (max-width: 576px) {
  .max-container {
    max-width: 540px;
  }
}
</style>
