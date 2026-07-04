<script setup lang="ts">
// 富文本编辑器：基于 wangEditor 5 核心 API（createEditor/createToolbar）
// 不使用 @wangeditor/editor-for-vue（其 `import default from "vue"` 与 Vue 3.5 ESM 不兼容）
// 主题适配见 styles/index.css 的 --w-e-* 变量（跟随 data-theme）
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { createEditor, createToolbar, type IDomEditor, type IToolbarConfig, type IEditorConfig } from '@wangeditor/editor'
import '@wangeditor/editor/dist/css/style.css'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  simple?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const toolbarRef = ref<HTMLDivElement | null>(null)
const editorRef = ref<HTMLDivElement | null>(null)
let editor: IDomEditor | null = null
let toolbar: ReturnType<typeof createToolbar> | null = null

// 简化工具栏（公告用），完整工具栏用默认
const toolbarConfig: Partial<IToolbarConfig> = props.simple
  ? { toolbarKeys: ['bold', 'italic', 'underline', 'bulletedList', 'numberedList', 'insertLink', 'undo', 'redo'] }
  : {}

const editorConfig: Partial<IEditorConfig> = {
  placeholder: props.placeholder || '请输入内容...',
  onChange: (editor: IDomEditor) => {
    const html = editor?.getHtml() || ''
    emit('update:modelValue', html)
  },
  MENU_CONF: {},
}

onMounted(() => {
  if (!editorRef.value || !toolbarRef.value) return
  editor = createEditor({
    selector: editorRef.value,
    html: props.modelValue || '',
    config: editorConfig,
    mode: 'default',
  })
  toolbar = createToolbar({
    editor,
    selector: toolbarRef.value,
    config: toolbarConfig,
    mode: 'default',
  })
})

// 外部 modelValue 变化时同步到编辑器（避免回环）
watch(() => props.modelValue, (v) => {
  if (editor && v !== editor.getHtml()) {
    editor.setHtml(v || '')
  }
})

onBeforeUnmount(() => {
  toolbar?.destroy()
  editor?.destroy()
  toolbar = null
  editor = null
})
</script>

<template>
  <div class="rich-editor">
    <div ref="toolbarRef" class="editor-toolbar"></div>
    <div ref="editorRef" :style="{ height: simple ? '180px' : '320px' }"></div>
  </div>
</template>
