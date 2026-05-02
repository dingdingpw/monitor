<script setup>
import highcharts from 'highcharts'
import moment from 'moment'
import {inject, onMounted, onUnmounted, ref, watch} from 'vue'

const props = defineProps({
  data: {
    type: Array
  }
})

const chartRef = ref()
const chart = ref(null)

const formatUnit = (value) => {
    return 'CPU: ' + (value).toFixed(2) + '%'
}

const options = ref({
  chart: {
    type: "area",
    style: {
      fontFamily: "Inter, -apple-system, BlinkMacSystemFont, Roboto, PingFang SC, Noto Sans CJK, WenQuanYi Micro Hei, Microsoft YaHei",
      fontSize: "12px",
      fontWeight: "bold",
      color: "white"
    },
    borderWidth: 0,
    // margin: [0, -8, 0, -8],
    backgroundColor: "#00000000"
  },
  title: {
    text: null
  },
  xAxis: {
    title: {
      text: null
    },
    type: "datetime",
  },
  yAxis: {
    title: {
      text: null
    },
  },
  tooltip: {
    formatter: function () {
      var d = new Date(this.x);
      var s = '<span style="font-weight: 400;font-size: 13px;">' + moment(d).format('HH:mm:ss') + '</span>';
      s += '<br/><span style="font-weight: 600;font-size: 15px;">' + formatUnit(this.point.y) + '</span>';
      return s;
    },
    backgroundColor: '#fff',
    borderColor: '#fafafa',
    borderRadius: 6,
    borderWidth: 1
  },
  plotOptions: {
    series: {
      boostThreshold: 1000,
      marker: {
        enabled: false,
        states: {
          hover: {
            enabled: true
          }
        }
      }
    }
  },
  series: [{
    color: '#3760d1',
    fillColor: '#3760d120',
    data: props.data.slice(-60),
    showInLegend: false,
    pointInterval: 1000
  }],
  credits: {
    enabled: false
  }
})

const updateOptions = (data) => {
  // const seriesData = [...options.value.series[0].data, data]
  chart.value.series[0].addPoint(data)
}

let interval = null

onMounted(() => {
  highcharts.setOptions({ global: { useUTC: false } });
  options.value.chart.renderTo = chartRef.value
  chart.value = highcharts.chart(options.value);

  interval = setInterval(() => {
    updateOptions(props.data[props.data.length - 1])
  }, 1000)
})

onUnmounted(() => {
  clearInterval(interval)
})

defineExpose({
  updateOptions,
  options
})
</script>

<template>
  <div class="name">CPU</div>
  <div ref="chartRef" class="card-bg-chart"></div>
</template>

<style scoped lang="scss">
.name {
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 600;
}
.card-bg-chart {
  width: 100%;
  height: 150px;
}
</style>