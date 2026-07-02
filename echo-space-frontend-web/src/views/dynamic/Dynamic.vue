<template>
  <div class="dynamic-page">
    <div v-if="!isLoggedIn" class="login-panel">
      <div class="login-title">登录后查看关注动态</div>
      <div class="login-tip">关注的博主发布视频或图片后，会出现在这里。</div>
      <el-button type="primary" @click="handleLogin">立即登录</el-button>
    </div>

    <div v-else class="dynamic-layout">
      <aside class="dynamic-sidebar">
        <section class="user-card" v-loading="loadingProfile">
          <div class="user-basic">
            <Avatar
              :avatar="currentUserInfo.avatar || loginStore.userInfo.avatar"
              :userId="currentUserId"
              :width="56"
              :lazy="false"
            ></Avatar>
            <div class="user-name" :title="currentUserInfo.nickName">
              {{ currentUserInfo.nickName || loginStore.userInfo.nickName }}
            </div>
          </div>
          <div class="user-counts">
            <div class="count-item">
              <div class="count-value">
                {{ formatCount(currentUserInfo.focusCount) }}
              </div>
              <div class="count-label">关注</div>
            </div>
            <div class="count-item">
              <div class="count-value">
                {{ formatCount(currentUserInfo.fansCount) }}
              </div>
              <div class="count-label">粉丝</div>
            </div>
            <div class="count-item">
              <div class="count-value">
                {{ formatCount(currentUserInfo.dynamicCount) }}
              </div>
              <div class="count-label">动态</div>
            </div>
          </div>
        </section>
        <section class="fixed-image-box"></section>
      </aside>

      <main class="dynamic-main">
        <section class="follow-panel">
          <div class="panel-title">关注的博主</div>
          <div class="follow-scroll" v-loading="loadingUsers">
            <button
              :class="['follow-item', selectedUserId == '' ? 'active' : '']"
              type="button"
              @click="handleSelectUser('')"
            >
              <div class="all-avatar">
                <div class="windmill-mark">
                  <span></span>
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
              <span class="follow-name">全部动态</span>
            </button>
            <button
              v-for="item in followUsers"
              :key="item.userId"
              :class="[
                'follow-item',
                selectedUserId == item.userId ? 'active' : '',
              ]"
              type="button"
              @click="handleSelectUser(item.userId)"
            >
              <Cover
                :source="item.avatar"
                defaultImg="user.png"
                borderRadius="50%"
                :width="52"
                :scale="1"
                :lazy="false"
              ></Cover>
              <span class="follow-name" :title="item.nickName">{{
                item.nickName
              }}</span>
            </button>
          </div>
        </section>

        <section
          class="feed-panel"
          v-loading="loadingFeed && feedList.length == 0"
        >
          <NoData
            v-if="!loadingFeed && feedList.length == 0"
            :msg="emptyFeedMsg"
          ></NoData>
          <article
            class="feed-card"
            v-for="item in feedList"
            :key="feedItemKey(item)"
          >
            <div class="feed-author">
              <Avatar
                :avatar="item.avatar"
                :userId="item.userId"
                :width="46"
                :lazy="false"
              ></Avatar>
              <div class="author-info">
                <router-link
                  class="nick-name"
                  :to="`/user/${item.userId}`"
                  target="_blank"
                >
                  {{ item.nickName }}
                </router-link>
                <div class="publish-time">
                  {{ formatTime(item.lastUpdateTime) }} 发布了{{ contentTypeName(item) }}
                </div>
              </div>
            </div>
            <div class="feed-content">
              <router-link
                class="video-title"
                :to="contentLink(item)"
                target="_blank"
              >
                {{ contentName(item) }}
              </router-link>
              <div class="video-introduction" v-if="item.introduction">
                {{ item.introduction }}
              </div>
              <router-link
                class="video-card"
                :to="contentLink(item)"
                target="_blank"
              >
                <div class="video-cover">
                  <Cover :source="contentCover(item)" fit="cover"></Cover>
                  <div class="play-time" v-if="!isImageContent(item)">{{ item.playTime }}</div>
                </div>
                <div class="video-info">
                  <div class="video-name">{{ contentName(item) }}</div>
                  <div class="video-desc">
                    {{ item.introduction || emptyContentDesc(item) }}
                  </div>
                  <div class="video-stats" v-if="!isImageContent(item)">
                    <span class="iconfont icon-play2">{{
                      formatCount(item.playCount)
                    }}</span>
                    <span class="iconfont icon-danmu">{{
                      formatCount(item.danmuCount)
                    }}</span>
                    <span class="iconfont icon-comment">{{
                      formatCount(item.commentCount)
                    }}</span>
                    <span class="iconfont icon-like-solid">{{
                      formatCount(item.likeCount)
                    }}</span>
                  </div>
                </div>
              </router-link>
            </div>
          </article>
        </section>

        <div class="load-more" v-if="feedList.length > 0">
          <el-button
            v-if="hasMore"
            :loading="loadingFeed"
            type="primary"
            plain
            @click="loadFeed"
          >
            加载更多
          </el-button>
          <div v-else class="reach-bottom">已经到底啦~~</div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, getCurrentInstance, onMounted, ref, watch } from 'vue'
import { useLoginStore } from '@/stores/loginStore.js'
import { useNavAction } from '@/stores/navActionStore'

const { proxy } = getCurrentInstance()
const loginStore = useLoginStore()
const navActionStore = useNavAction()

const currentUserInfo = ref({})
const followUsers = ref([])
const feedList = ref([])
const selectedUserId = ref('')
const nextCursor = ref('')
const hasMore = ref(false)
const loadingProfile = ref(false)
const loadingUsers = ref(false)
const loadingFeed = ref(false)
const hasLoaded = ref(false)
const CONTENT_TYPE_IMAGE = 1

const currentUserId = computed(() => loginStore.userInfo.userId || '')
const isLoggedIn = computed(() => currentUserId.value != '')
const emptyFeedMsg = computed(() => {
  if (followUsers.value.length == 0) {
    return '你还没有关注任何博主'
  }
  if (selectedUserId.value) {
    return '该博主暂无动态'
  }
  return '暂无动态'
})

onMounted(() => {
  navActionStore.setShowHeader(true)
  navActionStore.setFixedHeader(true)
  navActionStore.setFixedCategory(false)
  navActionStore.setShowCategory(false)
  navActionStore.setForceFixedHeader(false)

  if (!isLoggedIn.value) {
    handleLogin()
    return
  }
  initDynamicData()
})

watch(
  () => currentUserId.value,
  (newUserId, oldUserId) => {
    if (!newUserId) {
      clearDynamicData()
      return
    }
    if (newUserId != oldUserId) {
      initDynamicData(true)
    }
  }
)

const initDynamicData = async (force = false) => {
  if (hasLoaded.value && !force) {
    return
  }
  hasLoaded.value = true
  selectedUserId.value = ''
  await loadCurrentUserInfo()
  await loadFollowUsers()
  await resetFeed()
}

const clearDynamicData = () => {
  hasLoaded.value = false
  selectedUserId.value = ''
  nextCursor.value = ''
  hasMore.value = false
  currentUserInfo.value = {}
  followUsers.value = []
  feedList.value = []
}

const handleLogin = () => {
  loginStore.setLogin(true)
}

const loadCurrentUserInfo = async () => {
  loadingProfile.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadDynamicCurrentUserInfo,
  })
  loadingProfile.value = false
  if (!result) {
    return
  }
  currentUserInfo.value = result.data || {}
}

const loadFollowUsers = async () => {
  loadingUsers.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadDynamicFollowUsers,
  })
  loadingUsers.value = false
  if (!result) {
    return
  }
  followUsers.value = result.data || []
}

const handleSelectUser = (userId) => {
  if (selectedUserId.value == userId) {
    return
  }
  selectedUserId.value = userId
  resetFeed()
}

const resetFeed = async () => {
  nextCursor.value = ''
  hasMore.value = false
  feedList.value = []
  await loadFeed()
}

const loadFeed = async () => {
  if (loadingFeed.value) {
    return
  }
  if (!isLoggedIn.value) {
    handleLogin()
    return
  }

  loadingFeed.value = true
  const result = await proxy.Request({
    url: proxy.Api.loadDynamicFeed,
    params: {
      cursor: nextCursor.value,
      pageSize: 10,
      focusUserId: selectedUserId.value,
    },
  })
  loadingFeed.value = false
  if (!result) {
    return
  }

  const pageData = result.data || {}
  const list = pageData.list || []
  feedList.value = nextCursor.value ? feedList.value.concat(list) : list
  nextCursor.value = pageData.nextCursor || ''
  hasMore.value = pageData.hasMore === true
}

const formatTime = (time) => {
  return proxy.Utils.formatDate(time) || time || ''
}

const formatCount = (count) => {
  count = Number(count || 0)
  if (count >= 10000) {
    return `${(count / 10000).toFixed(1).replace('.0', '')}万`
  }
  return count
}

const normalizeContentType = (item) => Number(item?.contentType || 0)

const isImageContent = (item) => normalizeContentType(item) == CONTENT_TYPE_IMAGE

const contentId = (item) => item?.contentId || item?.videoId || ''

const contentName = (item) => item?.contentName || item?.videoName || ''

const contentCover = (item) => item?.contentCover || item?.videoCover || ''

const contentTypeName = (item) => (isImageContent(item) ? '图片' : '视频')

const contentLink = (item) => {
  const id = contentId(item)
  return isImageContent(item) ? `/gallery/${id}` : `/video/${id}`
}

const emptyContentDesc = (item) => (isImageContent(item) ? '这个图片暂时没有简介' : '这个视频暂时没有简介')

const feedItemKey = (item) => `${normalizeContentType(item)}_${contentId(item)}`
</script>

<style lang="scss" scoped>
.dynamic-page {
  width: 1200px;
  max-width: calc(100vw - 48px);
  margin: 20px auto 50px;
  min-height: 520px;
}

.login-panel {
  width: 760px;
  margin: 0px auto;
  background: #fff;
  border-radius: 6px;
  padding: 56px 20px;
  text-align: center;
  .login-title {
    font-size: 20px;
    color: var(--text);
    font-weight: 600;
  }
  .login-tip {
    margin: 12px 0px 24px;
    color: var(--text3);
  }
}

.dynamic-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.dynamic-main {
  min-width: 0;
}

.dynamic-sidebar {
  position: sticky;
  top: 84px;
  z-index: 2;
}

.user-card {
  background: #fff;
  border-radius: 6px;
  padding: 18px;
  min-height: 150px;
  .user-basic {
    display: flex;
    align-items: center;
    min-width: 0;
  }
  .user-name {
    flex: 1;
    min-width: 0;
    margin-left: 12px;
    color: var(--text);
    font-size: 16px;
    font-weight: 600;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .user-counts {
    margin-top: 18px;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    text-align: center;
    .count-value {
      color: var(--text);
      font-size: 16px;
      font-weight: 700;
      line-height: 22px;
    }
    .count-label {
      margin-top: 4px;
      color: var(--text3);
      font-size: 13px;
    }
  }
}

.fixed-image-box {
  margin-top: 12px;
  height: 260px;
  background: url('@/assets/dynamic/sidebar-banner.png') center / cover no-repeat;
  border: 1px solid #eef0f2;
  border-radius: 6px;
}

.follow-panel,
.feed-card {
  background: #fff;
  border-radius: 6px;
}

.follow-panel {
  padding: 18px 20px 16px;
  .panel-title {
    font-size: 16px;
    color: var(--text);
    font-weight: 600;
    margin-bottom: 14px;
  }
  .follow-scroll {
    min-height: 92px;
    display: flex;
    gap: 14px;
    overflow-x: auto;
    padding-bottom: 4px;
  }
  .follow-item {
    width: 70px;
    border: none;
    background: transparent;
    color: var(--text2);
    padding: 0px;
    cursor: pointer;
    flex: 0 0 auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    .follow-name {
      max-width: 70px;
      margin-top: 7px;
      font-size: 13px;
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
    &:hover,
    &.active {
      color: var(--blue);
    }
    &.active {
      .all-avatar,
      :deep(.image-panel) {
        box-shadow: 0 0 0 3px rgba(0, 174, 236, 0.2);
      }
    }
  }
}

.all-avatar {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: linear-gradient(135deg, #00aeec, #fb7299);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.windmill-mark {
  width: 24px;
  height: 24px;
  position: relative;
  &::before {
    content: '';
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
    position: absolute;
    left: 9px;
    top: 9px;
    z-index: 2;
  }
  span {
    width: 4px;
    height: 11px;
    border-radius: 4px 4px 1px 1px;
    background: currentColor;
    position: absolute;
    left: 10px;
    top: 1px;
    transform-origin: 2px 11px;
  }
  span:nth-child(2) {
    transform: rotate(90deg);
  }
  span:nth-child(3) {
    transform: rotate(180deg);
  }
  span:nth-child(4) {
    transform: rotate(270deg);
  }
}

.feed-panel {
  margin-top: 12px;
  min-height: 160px;
}

.feed-card {
  padding: 20px;
  margin-bottom: 12px;
  .feed-author {
    display: flex;
    align-items: center;
  }
  .author-info {
    margin-left: 12px;
    min-width: 0;
    .nick-name {
      color: var(--text);
      font-size: 16px;
      font-weight: 600;
      text-decoration: none;
      &:hover {
        color: var(--blue);
      }
    }
    .publish-time {
      margin-top: 5px;
      color: var(--text3);
      font-size: 13px;
    }
  }
  .feed-content {
    margin-left: 58px;
    margin-top: 12px;
  }
  .video-title {
    color: var(--text);
    font-size: 17px;
    line-height: 24px;
    font-weight: 600;
    text-decoration: none;
    word-break: break-word;
    &:hover {
      color: var(--blue);
    }
  }
  .video-introduction {
    margin-top: 8px;
    color: var(--text2);
    line-height: 22px;
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.video-card {
  margin-top: 12px;
  width: 100%;
  min-height: 132px;
  border: 1px solid #e3e5e7;
  border-radius: 6px;
  overflow: hidden;
  display: flex;
  text-decoration: none;
  color: var(--text2);
  background: #fff;
  &:hover {
    border-color: #b8e8f8;
  }
  .video-cover {
    width: 220px;
    height: 132px;
    flex: 0 0 auto;
    position: relative;
    background: #f1f2f3;
    .play-time {
      position: absolute;
      right: 8px;
      bottom: 6px;
      padding: 1px 5px;
      border-radius: 3px;
      background: rgba(0, 0, 0, 0.65);
      color: #fff;
      font-size: 12px;
    }
  }
  .video-info {
    flex: 1;
    min-width: 0;
    padding: 12px;
    .video-name {
      color: var(--text);
      font-weight: 600;
      line-height: 22px;
      display: -webkit-box;
      overflow: hidden;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
    }
    .video-desc {
      margin-top: 8px;
      color: var(--text3);
      font-size: 13px;
      line-height: 20px;
      display: -webkit-box;
      overflow: hidden;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
    }
    .video-stats {
      margin-top: 12px;
      display: flex;
      gap: 16px;
      color: var(--text3);
      font-size: 13px;
      .iconfont {
        &::before {
          margin-right: 4px;
          font-size: 15px;
        }
      }
    }
  }
}

.load-more {
  text-align: center;
  margin-top: 18px;
  .reach-bottom {
    color: var(--text3);
    line-height: 40px;
  }
}
</style>
