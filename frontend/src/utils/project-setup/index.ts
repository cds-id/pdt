import { generateIcons } from './iconGenerator'
import { generateWebManifest, WebManifestConfig } from './manifestGenerator'
import fs from 'fs'
import path from 'path'

/**
 * Project setup configuration
 */
export interface ProjectSetupConfig {
  /**
   * Project name
   */
  name: string
  /**
   * Short name for the project (used on homescreen)
   */
  shortName?: string
  /**
   * Project description
   */
  description: string
  /**
   * Path to the source icon (high resolution SVG or PNG)
   */
  iconSourcePath: string
  /**
   * Theme color for the application
   */
  themeColor?: string
  /**
   * Background color for the application
   */
  backgroundColor?: string
  /**
   * Public directory for assets
   * @default 'public'
   */
  publicDir?: string
}

/**
 * Sets up a new project with icons and web manifest
 */
export async function setupProject(config: ProjectSetupConfig): Promise<void> {
  const {
    name,
    shortName = name,
    description,
    iconSourcePath,
    themeColor = '#3b82f6', // Default to blue
    backgroundColor = '#ffffff',
    publicDir = 'public'
  } = config

  console.log(`\n🚀 Setting up project: ${name}\n`)

  // Ensure public directory exists
  const publicPath = path.resolve(process.cwd(), publicDir)
  if (!fs.existsSync(publicPath)) {
    fs.mkdirSync(publicPath, { recursive: true })
    console.log(`\u2705 Created public directory: ${publicDir}`)
  }

  // Generate icons
  const iconsDir = path.join(publicPath, 'icons')
  await generateIcons({
    sourcePath: iconSourcePath,
    outputDir: iconsDir,
    projectName: name.toLowerCase().replace(/\s+/g, '-')
  })

  // Generate web manifest
  const manifestConfig: WebManifestConfig = {
    name,
    shortName,
    description,
    iconsPath: '/icons',
    themeColor,
    backgroundColor,
    outputDir: publicPath
  }

  generateWebManifest(manifestConfig)

  // Update index.html with manifest tags
  updateIndexHtml(publicPath)

  console.log('\n✨ Project setup complete! ✨\n')
}

/**
 * Updates index.html with manifest meta tags
 */
function updateIndexHtml(publicDir: string): void {
  // Path to manifest meta tags snippet
  const snippetPath = path.join(publicDir, 'manifest-meta-tags.html')

  // Check if snippet exists
  if (!fs.existsSync(snippetPath)) {
    console.error('❌ Manifest meta tags snippet not found')
    return
  }

  const metaTags = fs.readFileSync(snippetPath, 'utf-8')

  // Find index.html (could be in public or root)
  let indexPath = path.join(process.cwd(), 'index.html')
  if (!fs.existsSync(indexPath)) {
    indexPath = path.join(process.cwd(), 'public', 'index.html')
  }
  if (!fs.existsSync(indexPath)) {
    console.error('❌ Could not find index.html')
    return
  }

  // Read index.html
  let indexHtml = fs.readFileSync(indexPath, 'utf-8')

  // Check if meta tags are already in index.html
  if (indexHtml.includes('<!-- Web App Manifest -->')) {
    console.log('⚠️ Manifest meta tags already exist in index.html')
    return
  }

  // Insert meta tags before </head>
  indexHtml = indexHtml.replace('</head>', `${metaTags}\n</head>`)

  // Write updated index.html
  fs.writeFileSync(indexPath, indexHtml)
  console.log('\u2705 Updated index.html with manifest meta tags')
}
