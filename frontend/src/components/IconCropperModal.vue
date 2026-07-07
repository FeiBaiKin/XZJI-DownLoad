<script setup lang="ts">
// 关键词图标上传流程（两步）：
//   1) 从媒体库选一张图
//   2) 用 cropperjs v1 在前端裁成方形圆角图标，传给后端 /admin/keywords/:id/icon
// 后端「保留裁剪后的图片，删除上传的原图」语义：媒体库原图不动，关键词的 icon 指向裁后图落盘文件。
import { ref, onBeforeUnmount, nextTick } from 'vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'
import { NModal, NButton, NIcon } from 'naive-ui'
import { X, Check, ArrowLeft } from '@vicons/tabler'
import { contentApi, type MediaFile } from '@/api/content'
import type { Keyword } from '@/types/api'
import MediaPicker from '@/components/MediaPicker.vue'

const props = defineProps<{
  show: boolean
  keyword: Keyword | null // 当前编辑的关键词
  /** 当前已设置的 icon url，用于回显 */
  currentIcon?: string
}>()
const emit = defineEmits<{
  close: []
  saved: [icon: string]
}>()

type Step = 'select' | 'crop'
const step = ref<Step>('select')
const pickedFile = ref<MediaFile | null>(null)
const cropperEl = ref<HTMLImageElement | null>(null)
let cropper: Cropper | null = null
const saving = ref(false)
const error = ref('')
const ICON_SIZE = 256 // 输出图标边长 px

function onOpen() {
  step.value = 'select'
  pickedFile.value = null
  error.value = ''
  destroyCropper()
}
function close() {
  destroyCropper()
  emit('close')
}

function onPick(file: MediaFile) {
  pickedFile.value = file
  step.value = 'crop'
  nextTick(() => initCropper())
}
function backToSelect() {
  step.value = 'select'
  destroyCropper()
}

function destroyCropper() {
  if (cropper) {
    cropper.destroy()
    cropper = null
  }
}

function initCropper() {
  if (!cropperEl.value || !pickedFile.value) return
  destroyCropper()
  const img = cropperEl.value
  img.src = pickedFile.value.url
  // 在 image load 后初始化
  cropper = new Cropper(img, {
    aspectRatio: 1,
    viewMode: 1,
    autoCropArea: 0.9,
    background: false,
    responsive: true,
    restore: false,
    movable: false,
    rotatable: false,
    scalable: false,
    zoomable: true,
    zoomOnTouch: true,
    zoomOnWheel: true,
    toggleDragModeOnDblclick: false,
  })
}

onBeforeUnmount(() => destroyCropper())

async function handleSave() {
  if (!cropper || !props.keyword) return
  if (!pickedFile.value) {
    error.value = '请先选择图片'
    return
  }
  saving.value = true
  error.value = ''
  const sourceFileId = pickedFile.value.id
  try {
    const canvas = cropper.getCroppedCanvas({
      width: ICON_SIZE,
      height: ICON_SIZE,
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high',
    })
    const blob: Blob = await new Promise((resolve, reject) => {
      canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('生成图标失败'))), 'image/png', 0.92)
    })
    const fd = new FormData()
    fd.append('file', blob, `icon_${props.keyword.id}.png`)
    const res = await contentApi.keywords.uploadIcon(props.keyword.id, fd)
    // 裁剪后删除原图：图片已经成功落盘为关键词图标，原图无需保留为冗余媒体
    // 失败不阻塞主流程，仅日志
    try {
      await contentApi.media.delete(sourceFileId)
    } catch (err) {
      console.warn('删除裁剪原图失败（已忽略）：', err)
    }
    emit('saved', res.icon)
    close()
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    error.value = e?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleClear() {
  if (!props.keyword) return
  saving.value = true
  error.value = ''
  try {
    await contentApi.keywords.clearIcon(props.keyword.id)
    emit('saved', '')
    close()
  } catch {
    error.value = '清除失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <NModal
      :show="show"
      :mask-closable="!saving"
      :close-on-esc="!saving"
      @update:show="(v: boolean) => { if (!v) close() }"
      @after-enter="onOpen"
    >
      <div class="w-[92vw] max-w-3xl overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
        <!-- 头部 -->
        <div class="flex items-center justify-between border-b border-zinc-100 px-5 py-4 dark:border-zinc-800">
          <div class="flex items-center gap-2">
            <button
              v-if="step === 'crop'"
              type="button"
              class="flex size-7 items-center justify-center rounded-full text-zinc-500 hover:bg-zinc-100 hover:text-zinc-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200 dark:focus-visible:ring-emerald-400/60"
              aria-label="返回选择图片"
              @click="backToSelect"
            >
              <NIcon :size="16"><ArrowLeft /></NIcon>
            </button>
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
              {{ step === 'select' ? '从媒体库选择图标' : '裁剪图标' }}
            </h3>
          </div>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-full text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-300 dark:focus-visible:ring-emerald-400/60"
            aria-label="关闭"
            @click="close"
          >
            <NIcon :size="16"><X /></NIcon>
          </button>
        </div>

        <!-- 当前图标预览（如果有） -->
        <div v-if="step === 'select' && currentIcon" class="border-b border-zinc-100 bg-zinc-50 px-5 py-3 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="flex items-center gap-3">
            <span class="text-xs text-zinc-500 dark:text-zinc-400">当前</span>
            <img :src="currentIcon" alt="当前图标" class="size-10 rounded-xl object-cover ring-1 ring-zinc-200 dark:ring-zinc-700" />
            <NButton size="small" quaternary type="error" :disabled="saving" @click="handleClear">清除当前图标</NButton>
          </div>
        </div>

        <!-- 提示条 -->
        <div v-if="error" class="mx-5 mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-500/10 dark:text-red-400">
          {{ error }}
        </div>

        <!-- 步骤 1：媒体库选择 -->
        <div v-if="step === 'select'" class="max-h-[60vh] overflow-y-auto p-5">
          <MediaPicker
            selectable
            @enter="onPick"
            @select="onPick"
            :page-size="18"
          />
          <p class="mt-3 text-center text-xs text-zinc-500 dark:text-zinc-400">双击或按 Enter 选中一张图继续裁剪</p>
        </div>

        <!-- 步骤 2：裁剪 -->
        <div v-else class="p-5">
          <p class="mb-3 text-sm text-zinc-500 dark:text-zinc-400">拖拽框选正方形图标区域，可滚轮缩放。输出为 {{ ICON_SIZE }}×{{ ICON_SIZE }} PNG。</p>
          <p class="mb-3 text-xs text-amber-600 dark:text-amber-400">保存后原图将被删除，仅保留裁剪后的图标。</p>
          <div class="overflow-hidden rounded-xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950/40">
            <img
              ref="cropperEl"
              class="max-h-[50vh] w-full"
              alt="待裁剪图片"
            />
          </div>
        </div>

        <!-- 底部操作 -->
        <div class="flex items-center justify-end gap-2 border-t border-zinc-100 px-5 py-3.5 dark:border-zinc-800">
          <NButton quaternary :disabled="saving" @click="close">取消</NButton>
          <NButton
            v-if="step === 'crop'"
            type="primary"
            size="large"
            :loading="saving"
            @click="handleSave"
          >
            <template #icon><NIcon><Check /></NIcon></template>
            保存图标
          </NButton>
        </div>
      </div>
    </NModal>
  </Teleport>
</template>