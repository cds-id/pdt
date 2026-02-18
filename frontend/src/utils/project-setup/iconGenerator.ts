import fs from 'fs'
import path from 'path'
import sharp from 'sharp'

/**
 * Configuration for icon generation
 */
interface IconGeneratorConfig {
  /**
   * Source icon path (should be high resolution, ideally SVG or at least 1024x1024 PNG)
   */
  sourcePath: string
  /**
   * Output directory for generated icons
   */
  outputDir: string
  /**
   * Project name (used for icon file naming)
   */
  projectName: string
  /**
   * Generate favicons
   * @default true
   */
  generateFavicons?: boolean
  /**
   * Generate iOS icons
   * @default true
   */
  generateIosIcons?: boolean
  /**
   * Generate Android icons
   * @default true
   */
  generateAndroidIcons?: boolean
}

/**
 * Generates a complete set of application icons from a source image
 * @param config Icon generator configuration
 */
export async function generateIcons(
  config: IconGeneratorConfig
): Promise<void> {
  const {
    sourcePath,
    outputDir,
    generateFavicons = true,
    generateIosIcons = true,
    generateAndroidIcons = true
  } = config

  // Create output directory if it doesn't exist
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true })
  }

  // Generate favicon.ico (multiple sizes in one file)
  if (generateFavicons) {
    const faviconDir = path.join(outputDir, 'favicon')
    if (!fs.existsSync(faviconDir)) {
      fs.mkdirSync(faviconDir, { recursive: true })
    }

    // Generate individual PNGs
    for (const size of [16, 32, 48, 64]) {
      await sharp(sourcePath)
        .resize(size, size)
        .png()
        .toFile(path.join(faviconDir, `favicon-${size}x${size}.png`))
    }

    // Note: Sharp doesn't directly support ICO format, so we'll generate PNGs instead
    // and inform the user that a favicon.ico can be created from these using a web service
    // or other tools if needed
    console.log('✅ Favicon PNG files generated')
    console.log('ℹ️ Note: Use these PNG files with a <link> tag in your HTML:')
    console.log(
      '   <link rel="icon" type="image/png" sizes="32x32" href="/favicon/favicon-32x32.png">'
    )
    console.log(
      '   <link rel="icon" type="image/png" sizes="16x16" href="/favicon/favicon-16x16.png">'
    )

    // Also generate a combined favicon manifest file to help with usage
    const faviconManifest = {
      icons: [16, 32, 48, 64].map((size) => ({
        size: `${size}x${size}`,
        path: `/favicon/favicon-${size}x${size}.png`
      }))
    }

    fs.writeFileSync(
      path.join(faviconDir, 'favicon-manifest.json'),
      JSON.stringify(faviconManifest, null, 2)
    )
  }

  // Generate standard app icons
  const standardIconSizes = [192, 512]
  for (const size of standardIconSizes) {
    await sharp(sourcePath)
      .resize(size, size)
      .png()
      .toFile(path.join(outputDir, `icon-${size}x${size}.png`))
    console.log(`✅ icon-${size}x${size}.png generated`)
  }

  // Generate Apple touch icons
  if (generateIosIcons) {
    const iosSizes = [120, 152, 167, 180]
    for (const size of iosSizes) {
      await sharp(sourcePath)
        .resize(size, size)
        .png()
        .toFile(path.join(outputDir, `apple-touch-icon-${size}x${size}.png`))
      console.log(`✅ apple-touch-icon-${size}x${size}.png generated`)
    }

    // Default Apple touch icon
    await sharp(sourcePath)
      .resize(180, 180)
      .png()
      .toFile(path.join(outputDir, 'apple-touch-icon.png'))
    console.log('✅ apple-touch-icon.png generated')
  }

  // Generate Android icons
  if (generateAndroidIcons) {
    const androidDir = path.join(outputDir, 'android')
    if (!fs.existsSync(androidDir)) {
      fs.mkdirSync(androidDir, { recursive: true })
    }

    const androidSizes = [
      { size: 36, name: 'ldpi' },
      { size: 48, name: 'mdpi' },
      { size: 72, name: 'hdpi' },
      { size: 96, name: 'xhdpi' },
      { size: 144, name: 'xxhdpi' },
      { size: 192, name: 'xxxhdpi' }
    ]

    for (const { size, name } of androidSizes) {
      await sharp(sourcePath)
        .resize(size, size)
        .png()
        .toFile(path.join(androidDir, `icon-${name}.png`))
      console.log(`✅ android/icon-${name}.png generated`)
    }
  }

  console.log('✅ All icons generated successfully!')
}
