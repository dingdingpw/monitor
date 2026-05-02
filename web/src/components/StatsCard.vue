<script setup>
import {formatBandwithBytes, formatBytes} from "../utils/utils.js";
import {inject} from "vue";

const handleChangeType = inject('handleChangeType')

const { stats, type } = defineProps({
  stats: {
    type: Object,
    default: () => ({
      total: 0,
      online: 0,
      offline: 0,
      bandwidth_up: 0,
      bandwidth_down: 0,
    }),
  },
  type: {
    type: String,
    default: "all",
  }
})

</script>

<template>

  <div class="hero">
    <a-row :gutter="20">
      <a-col :span="6" :xs="24" :sm="24" :md="6" :lg="6" :sl="6">
        <div class="hero-card all" :class="type === 'all' ? 'is-active' :''" @click="handleChangeType('all')">
          <div class="title">节点总览</div>
          <div class="value">
            <div class="status" style="background: #005fe7;"></div>
            <span class="num">{{stats.total}} {{ $t('server-unit') }}</span>
          </div>
        </div>
      </a-col>
      <a-col :span="6" :xs="24" :sm="24" :md="6" :lg="6" :sl="6">
        <div class="hero-card online" :class="type === 'online' ? 'is-active' :''" @click="handleChangeType('online')">
          <div class="title">在线节点</div>
          <div class="value">
            <div class="status" style="background: #1fb416;"></div>
            <span class="num">{{stats.online}} {{ $t('server-unit') }}</span>
          </div>
        </div>
      </a-col>
      <a-col :span="6" :xs="24" :sm="24" :md="6" :lg="6" :sl="6">
        <div class="hero-card offline" :class="type === 'offline' ? 'is-active' :''" @click="handleChangeType('offline')">
          <div class="title">离线节点</div>
          <div class="value">
            <div class="status" style="background: #b41616;"></div>
            <span class="num">{{stats.offline}} {{ $t('server-unit') }}</span>
          </div>
        </div>
      </a-col>
      <a-col :span="6" :xs="24" :sm="24" :md="6" :lg="6" :sl="6">
        <div class="hero-card">
          <div class="title">流量与带宽</div>
          <div class="value" style="display: block;">
            <div>
              {{ $t('traffic-info') }}
              <icon-arrow-up style="font-size: 14px;color: #d09453;" />
              <span style="font-size: 14px;color: #d09453;"> {{formatBytes(stats.traffic_up)}}</span>
              &nbsp;
              <icon-arrow-down style="font-size: 14px;color: #9a5fcd;" />
              <span style="font-size: 14px;color: #9a5fcd;">{{formatBytes(stats.traffic_down)}}</span>
            </div>
            <div>
              {{ $t('bandwidth-info') }}
              <icon-up-circle style="font-size: 14px;" />
              <span style="font-size: 14px;"> {{formatBandwithBytes(stats.bandwidth_up)}}</span>
              &nbsp;
              <icon-down-circle style="font-size: 14px;" />
              <span style="font-size: 14px;"> {{formatBandwithBytes(stats.bandwidth_down)}}</span>
            </div>
          </div>
        </div>
      </a-col>
    </a-row>
  </div>
</template>

<style scoped lang="scss">
.hero {
  margin: 24px 14px 0;

  .hero-card {
    margin-bottom: 20px;
    padding: 18px 22px;
    border: 1px solid rgba(23,33,47,.08);
    border-radius: 14px;
    background: #fff;
    min-height: 92px;
    box-shadow: none;
    cursor: pointer;
    transition: transform .2s ease, border-color .2s ease, box-shadow .2s ease;

    &.is-active,
    &:hover {
      &.all {
        border-color: #111827;
      }

      &.online {
        border-color: #079455;
      }

      &.offline {
        border-color: #d92d20;
      }
    }

    .title {
      margin-top: 6px;
      font-size: 12px;
      color: #6b7684;
      font-weight: 800;
      letter-spacing: .08em;
      margin-bottom: 14px;
    }

    .value {
      display: flex;
      align-items: center;
      .status {
        margin-right: 6px;
        width: 8px;
        height: 8px;
        border-radius: 10px;
        background: #333333;
      }

      .num {
        font-size: 24px;
        font-weight: 900;
      }
    }
  }
}

body[arco-theme='dark'] {
  .hero-card {
    border: 2px solid rgb(255 255 255 / 16%);
    box-shadow: none;
    background-color: #000000;
    color: #ffffff;
  }
}
</style>
