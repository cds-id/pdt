#!/usr/bin/env node

/**
 * Project Setup CLI
 *
 * This script helps to set up a new project with icons and web manifest.
 * It provides a command-line interface for the project setup utilities.
 */

import { setupProject } from '../src/utils/project-setup/index.js'
import { resolve } from 'path'
import fs from 'fs'
import prompts from 'prompts' // We'll need to add this dependency
import chalk from 'chalk' // We'll need to add this dependency

async function run() {
  console.log(chalk.bold.cyan('\n🚀 Project Setup Wizard 🚀\n'))
  console.log(
    chalk.gray(
      'This utility will help you set up your project with icons and web manifest.\n'
    )
  )

  const questions = [
    {
      type: 'text',
      name: 'name',
      message: 'What is the name of your project?',
      initial: 'My Awesome Project'
    },
    {
      type: 'text',
      name: 'shortName',
      message: 'Short name (for home screen):',
      initial: (prev) =>
        prev
          .split(' ')
          .map((word) => word[0])
          .join('')
    },
    {
      type: 'text',
      name: 'description',
      message: 'Project description:',
      initial: 'A React application built with Vite and Tailwind CSS'
    },
    {
      type: 'text',
      name: 'iconSourcePath',
      message: 'Path to source icon (SVG or high-res PNG):',
      initial: './src/assets/logo.svg',
      validate: (value) => {
        if (!fs.existsSync(resolve(process.cwd(), value))) {
          return `File not found: ${value}`
        }
        return true
      }
    },
    {
      type: 'text',
      name: 'themeColor',
      message: 'Theme color (hex):',
      initial: '#3b82f6',
      validate: (value) =>
        /^#[0-9A-Fa-f]{6}$/.test(value) || 'Please enter a valid hex color'
    },
    {
      type: 'text',
      name: 'backgroundColor',
      message: 'Background color (hex):',
      initial: '#ffffff',
      validate: (value) =>
        /^#[0-9A-Fa-f]{6}$/.test(value) || 'Please enter a valid hex color'
    },
    {
      type: 'text',
      name: 'publicDir',
      message: 'Public directory path:',
      initial: 'public'
    }
  ]

  try {
    const response = await prompts(questions, {
      onCancel: () => {
        console.log(chalk.yellow('\n⚠️ Setup canceled'))
        process.exit(0)
      }
    })

    console.log(
      chalk.gray('\nSetting up your project with the following configuration:')
    )
    console.log(
      chalk.gray('-----------------------------------------------------')
    )
    Object.entries(response).forEach(([key, value]) => {
      console.log(`${chalk.cyan(key)}: ${chalk.white(value)}`)
    })
    console.log(
      chalk.gray('-----------------------------------------------------\n')
    )

    await setupProject(response)

    // Final instructions
    console.log(chalk.green('\n✅ Project setup complete!'))
    console.log(chalk.white('\nNext steps:'))
    console.log(
      chalk.gray('1. Check your public directory for generated assets')
    )
    console.log(chalk.gray('2. Verify the manifest.json file'))
    console.log(
      chalk.gray('3. Make sure your index.html includes the meta tags')
    )
    console.log(chalk.gray('\nHappy coding! 🎉\n'))
  } catch (error) {
    console.error(chalk.red('\n❌ An error occurred during setup:'))
    console.error(chalk.red(error))
    process.exit(1)
  }
}

run()
