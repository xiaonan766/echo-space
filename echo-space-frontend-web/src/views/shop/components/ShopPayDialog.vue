<template>
  <el-dialog
    :model-value="modelValue"
    width="520px"
    class="shop-pay-dialog"
    :close-on-click-modal="false"
    @update:model-value="handleVisibleChange"
  >
    <template #header>
      <div class="dialog-title">硬币支付</div>
    </template>

    <div v-if="order" class="pay-content">
      <div class="order-summary">
        <div class="cover" :style="coverStyle">
          <span v-if="!order.coverUrl">暂无图片</span>
        </div>
        <div class="summary-info">
          <h3>{{ order.productName }}</h3>
          <div class="meta">{{ order.skuName }} · 数量 {{ order.quantity }}</div>
          <div class="order-no">订单号：{{ order.orderNo }}</div>
          <div :class="['status-tag', statusClass]">{{ statusText }}</div>
        </div>
      </div>

      <div class="coin-panel">
        <div class="coin-row">
          <span>当前硬币</span>
          <strong>{{ currentCoinCount }}</strong>
        </div>
        <div class="coin-row">
          <span>可用余额</span>
          <strong>{{ balanceAmountText }}</strong>
        </div>
        <div class="coin-row">
          <span>订单金额</span>
          <strong class="price">{{ amountText }}</strong>
        </div>
        <div class="coin-row">
          <span>需要硬币</span>
          <strong>{{ requiredCoins }}</strong>
        </div>
      </div>

      <div v-if="disabledReason" class="pay-tip">{{ disabledReason }}</div>
      <div v-else class="pay-tip success">将扣除 {{ requiredCoins }} 个硬币完成支付。</div>
    </div>

    <NoData v-else msg="暂无待支付订单"></NoData>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleVisibleChange(false)">取消</el-button>
        <el-button plain :disabled="refreshing" @click="$emit('refresh')">刷新状态</el-button>
        <el-button
          type="primary"
          class="pay-button"
          :loading="paying"
          :disabled="payDisabled"
          @click="$emit('confirm')"
        >
          确认支付
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, getCurrentInstance } from 'vue'
import { useLoginStore } from '@/stores/loginStore.js'

const { proxy } = getCurrentInstance()
const loginStore = useLoginStore()

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  order: {
    type: Object,
    default: null,
  },
  paying: {
    type: Boolean,
    default: false,
  },
  refreshing: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue', 'confirm', 'refresh'])

const currentCoinCount = computed(() => {
  return Number(loginStore.userInfo?.currentCoinCount || 0)
})

const payAmount = computed(() => {
  return Number(props.order?.payAmount ?? props.order?.totalAmount ?? 0)
})

const requiredCoins = computed(() => {
  return Math.round(payAmount.value * 100)
})

const balanceAmountText = computed(() => {
  return formatMoney(currentCoinCount.value / 100)
})

const amountText = computed(() => {
  return props.order?.totalAmountText || formatMoney(payAmount.value)
})

const isWaitPay = computed(() => {
  return props.order?.orderStatus === 1 && Number(props.order?.payStatus || 0) === 0
})

const isPaid = computed(() => {
  return props.order?.orderStatus === 3 || props.order?.payStatus === 1
})

const isCoinInsufficient = computed(() => {
  return currentCoinCount.value < requiredCoins.value
})

const payDisabled = computed(() => {
  return props.paying || !props.order || !isWaitPay.value || isCoinInsufficient.value
})

const disabledReason = computed(() => {
  if (!props.order) {
    return '订单不存在，请刷新后重试。'
  }
  if (props.order.orderStatus === 0) {
    return '库存仍在锁定中，请稍后刷新状态。'
  }
  if (isPaid.value) {
    return '订单已支付，无需重复支付。'
  }
  if (props.order.orderStatus === 2) {
    return '抢购失败，无法继续支付。'
  }
  if (!isWaitPay.value) {
    return '当前订单状态暂不支持支付。'
  }
  if (isCoinInsufficient.value) {
    return `余额不足，还差 ${requiredCoins.value - currentCoinCount.value} 个硬币。`
  }
  return ''
})

const statusText = computed(() => {
  if (props.order?.orderStatusName) {
    return props.order.orderStatusName
  }
  if (props.order?.orderStatus === 0) {
    return '库存锁定中'
  }
  if (props.order?.orderStatus === 1) {
    return '待支付'
  }
  if (props.order?.orderStatus === 2) {
    return '抢购失败'
  }
  if (props.order?.orderStatus === 3) {
    return '已支付'
  }
  return '未知状态'
})

const statusClass = computed(() => {
  if (props.order?.orderStatus === 0) {
    return 'locking'
  }
  if (props.order?.orderStatus === 1) {
    return 'wait-pay'
  }
  if (props.order?.orderStatus === 2) {
    return 'failed'
  }
  if (props.order?.orderStatus === 3) {
    return 'paid'
  }
  return ''
})

const coverStyle = computed(() => {
  const coverUrl = props.order?.coverUrl
  if (!coverUrl) {
    return {}
  }
  return {
    backgroundImage: `url(${buildImageUrl(coverUrl)})`,
  }
})

const handleVisibleChange = (visible) => {
  emit('update:modelValue', visible)
}

const buildImageUrl = (source) => {
  if (!source) {
    return ''
  }
  if (/^(https?:)?\/\//.test(source) || source.startsWith('data:') || source.startsWith('/')) {
    return source
  }
  return `${proxy.Api.sourcePath}${source}`
}

const formatMoney = (value) => {
  const numberValue = Number(value || 0)
  return `¥${numberValue.toFixed(2).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1')}`
}
</script>

<style lang="scss" scoped>
.dialog-title {
  color: #222;
  font-size: 18px;
  font-weight: 700;
}

.pay-content {
  color: #222;
}

.order-summary {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid #eef0f3;
}

.cover {
  width: 92px;
  height: 92px;
  border-radius: 6px;
  background-color: #edf0f5;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  font-size: 13px;
  overflow: hidden;
}

.summary-info {
  min-width: 0;
  h3 {
    margin: 0 0 8px;
    font-size: 18px;
    line-height: 26px;
    word-break: break-word;
  }
}

.meta,
.order-no {
  color: #909399;
  line-height: 24px;
}

.status-tag {
  display: inline-flex;
  margin-top: 8px;
  padding: 3px 9px;
  border-radius: 3px;
  color: #909399;
  border: 1px solid #c8c9cc;
  &.locking {
    color: #e6a23c;
    border-color: #e6a23c;
  }
  &.wait-pay {
    color: #fb7299;
    border-color: #fb7299;
  }
  &.paid {
    color: #67c23a;
    border-color: #67c23a;
  }
}

.coin-panel {
  margin-top: 18px;
  padding: 16px 18px;
  border-radius: 6px;
  background: #f7f8fa;
}

.coin-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 30px;
  color: #61666d;
  strong {
    color: #222;
    font-size: 16px;
  }
  .price {
    color: #fb7299;
  }
}

.pay-tip {
  margin-top: 14px;
  color: #f56c6c;
  line-height: 22px;
  &.success {
    color: #67c23a;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.pay-button {
  border-color: #fb7299;
  background: #fb7299;
  &:hover,
  &:focus {
    border-color: #ff8aac;
    background: #ff8aac;
  }
}
</style>
