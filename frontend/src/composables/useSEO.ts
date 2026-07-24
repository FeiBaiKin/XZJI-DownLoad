import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useSiteSettingsStore } from '@/stores/siteSettings'

const DEFAULT_SITE_NAME = 'XZJI'
const ADMIN_TITLE_SUFFIX = '后台'

const meta = (selector: string, attrs: Record<string, string>) => {
  let el = document.head.querySelector<HTMLMetaElement>(selector)
  if (!el) {
    el = document.createElement('meta')
    document.head.appendChild(el)
  }
  Object.entries(attrs).forEach(([key, value]) => el!.setAttribute(key, value))
}

function isAdminRoute(path: string) {
  return path === '/admin' || path.startsWith('/admin/')
}

function getSiteName(siteName: string) {
  return siteName || DEFAULT_SITE_NAME
}

function getDocumentTitle(route: ReturnType<typeof useRoute>, siteName: string, titleTemplate: string) {
  const baseName = getSiteName(siteName)

  // 后台及后台登录页使用独立标题；前台标题规则保持原样。
  if (isAdminRoute(route.path)) return `${baseName}${ADMIN_TITLE_SUFFIX}`

  const page = route.name === 'categories' ? '分类浏览' : ''
  return page
    ? (titleTemplate || '%page% | %site%')
      .replace('%page%', page)
      .replace('%site%', baseName)
    : baseName
}

export function useSEO() {
  const route = useRoute()
  const site = useSiteSettingsStore()

  const syncTitle = () => {
    document.title = getDocumentTitle(route, site.siteName, site.seoTitleTemplate)
  }

  const syncSEO = () => {
    syncTitle()
    if (!site.loaded) return

    const title = document.title
    const origin = (site.seoSiteUrl || location.origin).replace(/\/$/, '')
    const canonical = origin + route.path
    meta('meta[name=description]', { name: 'description', content: site.siteDescription })
    meta('meta[name=keywords]', { name: 'keywords', content: site.seoDefaultKeywords })
    meta('meta[name=robots]', {
      name: 'robots',
      content: site.seoEnabled && site.seoRobotsIndex
        ? 'index, follow, max-image-preview:large, max-snippet:-1'
        : 'noindex, nofollow',
    })
    for (const [property, value] of Object.entries({
      'og:title': title,
      'og:description': site.siteDescription,
      'og:type': 'website',
      'og:url': canonical,
    })) {
      meta(`meta[property='${property}']`, { property, content: value })
    }
    meta(`meta[name='twitter:card']`, { name: 'twitter:card', content: 'summary_large_image' })
    if (site.seoDefaultImage) {
      meta(`meta[property='og:image']`, { property: 'og:image', content: site.seoDefaultImage })
    }
    if (site.seoGoogleVerification) {
      meta(`meta[name='google-site-verification']`, {
        name: 'google-site-verification',
        content: site.seoGoogleVerification,
      })
    }
    if (site.seoBingVerification) {
      meta(`meta[name='msvalidate.01']`, {
        name: 'msvalidate.01',
        content: site.seoBingVerification,
      })
    }

    let link = document.head.querySelector<HTMLLinkElement>('link[rel=canonical]')
    if (!link) {
      link = document.createElement('link')
      link.rel = 'canonical'
      document.head.appendChild(link)
    }
    link.href = canonical

    document.getElementById('site-structured-data')?.remove()
    if (site.geoEnabled) {
      const script = document.createElement('script')
      script.id = 'site-structured-data'
      script.type = 'application/ld+json'
      script.textContent = JSON.stringify({
        '@context': 'https://schema.org',
        '@graph': [
          {
            '@type': 'WebSite',
            url: origin + '/',
            name: site.siteName,
            description: site.siteDescription,
            inLanguage: 'zh-CN',
          },
          {
            '@type': 'Organization',
            name: site.geoOrganizationName || site.siteName,
            url: origin + '/',
            logo: site.geoOrganizationLogo || site.siteLogoUrl || undefined,
            description: site.geoOrganizationDescription || site.siteDescription,
            email: site.geoContactEmail || undefined,
            sameAs: site.geoSocialProfiles.split('\n').map((value) => value.trim()).filter(Boolean),
          },
        ],
      })
      document.head.appendChild(script)
    }
  }

  // 标题需要在设置加载前也能区分后台；设置变更后同步更新标题。
  watch(
    [
      () => route.fullPath,
      () => route.name,
      () => site.siteName,
      () => site.seoTitleTemplate,
      () => site.loaded,
    ],
    syncSEO,
    { immediate: true },
  )
}
