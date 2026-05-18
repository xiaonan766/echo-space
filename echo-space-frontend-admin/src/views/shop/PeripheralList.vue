<template>
  <div class="top-panel">
    <el-card>
      <el-form :model="searchForm" label-width="80px" @submit.prevent>
        <el-row :gutter="10">
          <el-col :span="5">
            <el-form-item label="商品名称">
              <el-input
                v-model="searchForm.productNameFuzzy"
                clearable
                placeholder="输入商品名称搜索"
                @keyup.enter="handleSearch"
              ></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="上架状态">
              <el-select v-model="searchForm.status" clearable placeholder="全部">
                <el-option :value="1" label="已上架"></el-option>
                <el-option :value="0" label="已下架"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="销售状态">
              <el-select v-model="searchForm.saleStatus" clearable placeholder="全部">
                <el-option :value="0" label="待开售"></el-option>
                <el-option :value="1" label="在售"></el-option>
                <el-option :value="2" label="已售罄"></el-option>
                <el-option :value="3" label="已下架"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-button type="success" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
            <el-button type="primary" @click="showEdit()">新增周边</el-button>
          </el-col>
        </el-row>
      </el-form>
    </el-card>
  </div>
  <el-card class="table-data-card">
    <Table
      ref="tableInfoRef"
      :columns="columns"
      :fetch="loadDataList"
      :dataSource="tableData"
      :options="tableOptions"
      :extHeight="tableOptions.extHeight"
    >
      <template #cover="{ row }">
        <div class="cover">
          <Cover :source="row.coverUrl" fit="cover" :preview="true"></Cover>
        </div>
      </template>

      <template #productInfo="{ row }">
        <div class="product-info">
          <div class="product-name">{{ row.productName }}</div>
          <div class="description">{{ row.description || '暂无描述' }}</div>
          <div class="time">开售：{{ row.saleStartTime || '立即开售' }}</div>
        </div>
      </template>

      <template #price="{ row }"> {{ row.priceText || formatPrice(row.price) }} </template>

      <template #stock="{ row }">
        <div class="stock-info">
          <span>总 {{ row.totalStock }}</span>
          <span>可售 {{ row.availableStock }}</span>
          <span>锁定 {{ row.lockedStock }}</span>
          <span>已售 {{ row.soldStock }}</span>
        </div>
      </template>

      <template #status="{ row }">
        <el-tag :type="row.status == 1 ? 'success' : 'info'">
          {{ row.statusName }}
        </el-tag>
      </template>

      <template #saleStatus="{ row }">
        <el-tag :type="saleStatusTypeMap[row.saleStatus] || 'info'">
          {{ row.saleStatusName }}
        </el-tag>
      </template>

      <template #recommendStatus="{ row }">
        {{ row.recommendStatus == 1 ? '已推荐' : '未推荐' }}
      </template>

      <template #slotOperation="{ row }">
        <div class="row-op-panel">
          <a class="a-link" href="javascript:void(0)" @click="showEdit(row)">编辑</a>
          <el-divider direction="vertical" />
          <a class="a-link" href="javascript:void(0)" @click="changeStatus(row)">
            {{ row.status == 1 ? '下架' : '上架' }}
          </a>
        </div>
      </template>
    </Table>
  </el-card>
  <PeripheralEdit ref="peripheralEditRef" @reload="loadDataList"></PeripheralEdit>
</template>

<script setup>
import PeripheralEdit from './PeripheralEdit.vue'
import { getCurrentInstance, ref } from 'vue'

const { proxy } = getCurrentInstance()

const searchForm = ref({})
const tableData = ref({})
const tableInfoRef = ref()
const tableOptions = ref({
  extHeight: 0,
})

const saleStatusTypeMap = {
  0: 'warning',
  1: 'success',
  2: 'danger',
  3: 'info',
}

const columns = [
  {
    label: '封面',
    prop: 'coverUrl',
    width: 110,
    scopedSlots: 'cover',
  },
  {
    label: '商品信息',
    prop: 'productName',
    scopedSlots: 'productInfo',
  },
  {
    label: '价格',
    prop: 'price',
    width: 110,
    scopedSlots: 'price',
  },
  {
    label: '库存',
    prop: 'stock',
    width: 230,
    scopedSlots: 'stock',
  },
  {
    label: '上架状态',
    prop: 'status',
    width: 100,
    scopedSlots: 'status',
  },
  {
    label: '销售状态',
    prop: 'saleStatus',
    width: 100,
    scopedSlots: 'saleStatus',
  },
  {
    label: '推荐',
    prop: 'recommendStatus',
    width: 90,
    scopedSlots: 'recommendStatus',
  },
  {
    label: '排序',
    prop: 'sort',
    width: 80,
  },
  {
    label: '更新时间',
    prop: 'updateTime',
    width: 170,
  },
  {
    label: '操作',
    prop: 'operation',
    width: 130,
    scopedSlots: 'slotOperation',
  },
]

const loadDataList = async () => {
  let params = {
    pageNo: tableData.value.pageNo,
    pageSize: tableData.value.pageSize,
  }
  Object.assign(params, searchForm.value)
  let result = await proxy.Request({
    url: proxy.Api.loadPeripheral,
    params,
  })
  if (!result) {
    return
  }
  Object.assign(tableData.value, result.data)
}

const handleSearch = () => {
  tableData.value.pageNo = 1
  loadDataList()
}

const handleReset = () => {
  searchForm.value = {}
  handleSearch()
}

const peripheralEditRef = ref()
const showEdit = (row = {}) => {
  peripheralEditRef.value.showEdit(row)
}

const changeStatus = (row) => {
  const nextStatus = row.status == 1 ? 0 : 1
  const actionName = nextStatus == 1 ? '上架' : '下架'
  proxy.Confirm({
    message: `确定要${actionName}【${row.productName}】吗？`,
    okfun: async () => {
      let result = await proxy.Request({
        url: proxy.Api.changePeripheralStatus,
        params: {
          productId: row.productId,
          status: nextStatus,
        },
      })
      if (!result) {
        return
      }
      proxy.Message.success(`${actionName}成功`)
      loadDataList()
    },
  })
}

const formatPrice = (price) => {
  return `¥${Number(price || 0).toFixed(2)}`
}
</script>

<style lang="scss" scoped>
.cover {
  width: 82px;
  height: 82px;
}
.product-info {
  min-width: 0;
  .product-name {
    color: var(--text);
    font-weight: 600;
  }
  .description {
    margin-top: 6px;
    color: var(--text3);
    line-height: 20px;
    max-width: 520px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  .time {
    margin-top: 6px;
    color: var(--text3);
    font-size: 12px;
  }
}
.stock-info {
  display: grid;
  grid-template-columns: repeat(2, minmax(70px, 1fr));
  row-gap: 6px;
  color: var(--text2);
  font-size: 13px;
}
</style>
