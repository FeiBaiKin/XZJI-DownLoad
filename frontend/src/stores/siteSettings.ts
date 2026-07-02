import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

export const useSiteSettingsStore = defineStore('siteSettings', () => {
  const siteName = ref('XZJI')
  const siteDescription = ref('')
  const footerText = ref('')
  const siteLogoType = ref('text')
  const siteLogoUrl = ref('')
  const loaded = ref(false)

  async function fetch() {
    if (loaded.value) return
    try {
      const { data } = await axios.get('/api/settings')
      siteName.value = data.siteName || 'XZJI'
      siteDescription.value = data.siteDescription || ''
      footerText.value = data.footerText || ''
      siteLogoType.value = data.siteLogoType || 'text'
      siteLogoUrl.value = data.siteLogoUrl || ''
      loaded.value = true
    } catch {
      // use defaults
    }
  }

  return { siteName, siteDescription, footerText, siteLogoType, siteLogoUrl, loaded, fetch }
})
