import fs from 'fs'
import path from 'path'

/**
 * Web App Manifest configuration options
 */
export interface WebManifestConfig {
  /**
   * The name of the application
   */
  name: string
  /**
   * A short name for the application (used on homescreen)
   */
  shortName: string
  /**
   * Description of the application
   */
  description: string
  /**
   * The base URL path for the icons
   */
  iconsPath: string
  /**
   * Theme color for the application
   * @default '#ffffff'
   */
  themeColor?: string
  /**
   * Background color for the splash screen
   * @default '#ffffff'
   */
  backgroundColor?: string
  /**
   * Display mode for the application
   * @default 'standalone'
   */
  display?: 'fullscreen' | 'standalone' | 'minimal-ui' | 'browser'
  /**
   * The orientation of the application
   * @default 'portrait'
   */
  orientation?: 'portrait' | 'landscape' | 'any'
  /**
   * The scope of the application
   * @default '/'
   */
  scope?: string
  /**
   * The start URL of the application
   * @default '/'
   */
  startUrl?: string
  /**
   * Output directory for the manifest file
   */
  outputDir: string
  /**
   * Filename for the manifest
   * @default 'manifest.json'
   */
  filename?: string
}

/**
 * Generates a web app manifest file based on the provided configuration
 * @param config Configuration for the web app manifest
 */
export function generateWebManifest(config: WebManifestConfig): void {
  const {
    name,
    shortName,
    description,
    iconsPath,
    themeColor = '#ffffff',
    backgroundColor = '#ffffff',
    display = 'standalone',
    orientation = 'portrait',
    scope = '/',
    startUrl = '/',
    outputDir,
    filename = 'manifest.json'
  } = config

  // Create icons definition for the manifest
  const icons = [
    {
      src: `${iconsPath}/icon-192x192.png`,
      sizes: '192x192',
      type: 'image/png',
      purpose: 'any maskable'
    },
    {
      src: `${iconsPath}/icon-512x512.png`,
      sizes: '512x512',
      type: 'image/png',
      purpose: 'any maskable'
    }
  ]

  // Apple touch icons (not directly used in manifest but should be included in HTML)
  const appleTouchIcons = [
    {
      sizes: '180x180',
      href: `${iconsPath}/apple-touch-icon.png`
    },
    {
      sizes: '152x152',
      href: `${iconsPath}/apple-touch-icon-152x152.png`
    },
    {
      sizes: '167x167',
      href: `${iconsPath}/apple-touch-icon-167x167.png`
    },
    {
      sizes: '120x120',
      href: `${iconsPath}/apple-touch-icon-120x120.png`
    }
  ]

  // Create manifest object
  const manifest = {
    name,
    short_name: shortName,
    description,
    icons,
    theme_color: themeColor,
    background_color: backgroundColor,
    display,
    orientation,
    scope,
    start_url: startUrl
  }

  // Ensure output directory exists
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true })
  }

  // Write manifest file
  fs.writeFileSync(
    path.join(outputDir, filename),
    JSON.stringify(manifest, null, 2)
  )
  console.log(`\u2705 Web App Manifest generated at ${filename}`)

  // Generate HTML snippet for inclusion in index.html
  const htmlSnippet = generateManifestHtmlSnippet({
    manifestPath: `/${filename}`,
    themeColor,
    appleTouchIcons
  })

  fs.writeFileSync(path.join(outputDir, 'manifest-meta-tags.html'), htmlSnippet)
  console.log('\u2705 HTML snippet for manifest integration generated')
}

interface HtmlSnippetConfig {
  manifestPath: string
  themeColor: string
  appleTouchIcons: Array<{ sizes: string; href: string }>
}

/**
 * Generates HTML snippet for including the web app manifest and related meta tags
 */
function generateManifestHtmlSnippet(config: HtmlSnippetConfig): string {
  const { manifestPath, themeColor, appleTouchIcons } = config

  let htmlSnippet = `<!-- Web App Manifest -->\n<link rel="manifest" href="${manifestPath}">\n\n`
  htmlSnippet += `<!-- Theme Color -->\n<meta name="theme-color" content="${themeColor}">\n\n`
  htmlSnippet += `<!-- iOS Meta Tags -->\n<meta name="apple-mobile-web-app-capable" content="yes">\n`
  htmlSnippet += `<meta name="apple-mobile-web-app-status-bar-style" content="default">\n`
  htmlSnippet += `<meta name="apple-mobile-web-app-title" content="${
    manifestPath.split('/').pop() || 'App'
  }">\n\n`

  // Add Apple touch icon links
  htmlSnippet += `<!-- Apple Touch Icons -->\n`
  appleTouchIcons.forEach((icon) => {
    htmlSnippet += `<link rel="apple-touch-icon" sizes="${icon.sizes}" href="${icon.href}">\n`
  })
  htmlSnippet += `<link rel="apple-touch-icon" href="${appleTouchIcons[0].href}">\n\n`

  // Add favicon links
  htmlSnippet += `<!-- Favicons -->\n`
  htmlSnippet += `<link rel="icon" type="image/png" sizes="32x32" href="/favicon/favicon-32x32.png">\n`
  htmlSnippet += `<link rel="icon" type="image/png" sizes="16x16" href="/favicon/favicon-16x16.png">\n`
  htmlSnippet += `<link rel="shortcut icon" href="/favicon.ico">\n`

  return htmlSnippet
}
