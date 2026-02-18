import fs from 'fs'
import { createInterface } from 'readline'

const rl = createInterface({
  input: process.stdin,
  output: process.stdout
})

const question = (query) =>
  new Promise((resolve) => rl.question(query, resolve))

async function setupTemplate() {
  try {
    // Read package.json
    const packageJson = JSON.parse(fs.readFileSync('./package.json', 'utf8'))

    // Get user input
    console.log('🚀 Welcome to the template setup!\n')

    const projectName = await question('📦 Project name: ')
    const description = await question('📝 Project description: ')
    const authorName = await question('👤 Author name: ')
    const authorEmail = await question('📧 Author email: ')
    const githubUsername = await question('🐙 GitHub username: ')

    // Update package.json
    packageJson.name = projectName
    packageJson.description = description
    packageJson.author = {
      name: authorName,
      email: authorEmail,
      url: `https://github.com/${githubUsername}`
    }
    packageJson.bugs = {
      url: `https://github.com/${githubUsername}/${projectName}/issues`,
      email: authorEmail
    }

    // Write updated package.json
    fs.writeFileSync(
      './package.json',
      JSON.stringify(packageJson, null, 2),
      'utf8'
    )

    // Update index.html title
    const indexPath = './index.html'
    let indexContent = fs.readFileSync(indexPath, 'utf8')

    // Replace the title tag content
    indexContent = indexContent.replace(
      /<title>.*?<\/title>/,
      `<title>${projectName}</title>`
    )

    // Write updated index.html
    fs.writeFileSync(indexPath, indexContent, 'utf8')

    console.log('\n✅ Template setup completed successfully!')
    console.log('🎉 You can now start developing your project\n')
    console.log('Next steps:')
    console.log('1. npm install')
    console.log('2. git init')
    console.log('3. npm run dev')

    rl.close()
  } catch (error) {
    console.error('❌ Error setting up template:', error)
    process.exit(1)
  }
}

setupTemplate()
