import { Link } from 'react-router-dom'

export function LandingPage() {
  return (
    <div className="bg-[#F8C630] min-h-screen">
      {/* Navigation */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-[#F8C630]/90 backdrop-blur-md border-b border-[#1B1B1E]/20">
        <div className="w-full px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <span className="text-xl font-bold text-[#1B1B1E]">PDT</span>
              <span className="ml-2 text-sm text-[#1B1B1E]/70">Personal Development Tracker</span>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                to="/login"
                className="text-[#1B1B1E] hover:text-[#96031A] transition-colors font-medium"
              >
                Login
              </Link>
              <Link
                to="/register"
                className="px-4 py-2 bg-[#1B1B1E] text-[#FBFFFE] rounded-lg hover:bg-[#96031A] transition-colors font-medium"
              >
                Get Started
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-24 pb-20 px-4">
        <div className="w-full max-w-7xl mx-auto text-center">
          <h1 className="text-5xl md:text-6xl font-bold text-[#1B1B1E] mb-6">
            Your Personal
            <span className="text-[#96031A]"> Development Tracker</span>
          </h1>
          <p className="text-xl text-[#1B1B1E]/80 mb-8 max-w-2xl mx-auto">
            Automatically track commits across GitHub & GitLab, link Jira cards,
            and generate daily reports. Stay on top of your development work.
          </p>
          <div className="flex justify-center gap-4">
            <Link
              to="/register"
              className="px-8 py-3 bg-[#96031A] text-[#FBFFFE] font-semibold rounded-lg hover:bg-[#96031A]/80 transition-colors"
            >
              Get Started Free
            </Link>
            <Link
              to="/login"
              className="px-8 py-3 border-2 border-[#1B1B1E] text-[#1B1B1E] font-semibold rounded-lg hover:bg-[#1B1B1E] hover:text-[#FBFFFE] transition-colors"
            >
              View Demo
            </Link>
          </div>
        </div>
      </section>

      {/* Integrations Section */}
      <section className="py-20 bg-[#FBFFFE]">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            Seamless Integrations
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            {/* GitHub */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-[#1B1B1E] rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">GitHub</h3>
              <p className="text-slate-600">Track commits from your GitHub repositories automatically.</p>
            </div>

            {/* GitLab */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-orange-500 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 4.82 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0 1 18.6 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.51L23 13.45a.84.84 0 0 1-.35.94z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">GitLab</h3>
              <p className="text-slate-600">Sync commits from your GitLab projects seamlessly.</p>
            </div>

            {/* Jira */}
            <div className="p-6 border border-slate-200 rounded-xl hover:shadow-lg transition-shadow">
              <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mb-4">
                <svg className="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M11.571 11.513H0a5.218 5.218 0 0 0 5.232 5.215h2.13v2.057A5.215 5.215 0 0 0 12.575 24V12.518a1.005 1.005 0 0 0-1.005-1.005zm5.723-5.756H5.436a5.215 5.215 0 0 0 5.215 5.214h2.129v2.058a5.218 5.218 0 0 0 5.215 5.214V6.758a1.001 1.001 0 0 0-1.001-1.001zM23.013 0H11.455a5.215 5.215 0 0 0 5.215 5.215h2.129v2.057A5.215 5.215 0 0 0 24 12.483V1.005A1.005 1.005 0 0 0 23.013 0z"/>
                </svg>
              </div>
              <h3 className="text-xl font-semibold text-slate-900 mb-2">Jira</h3>
              <p className="text-slate-600">Link commits to Jira cards and track your sprint progress.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-[#FBFFFE]">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            Everything You Need to Track Your Work
          </h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            <div className="text-center">
              <div className="w-14 h-14 bg-[#96031A]/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-[#96031A]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Automatic Sync</h3>
              <p className="text-slate-600 text-sm">Commits synced automatically from GitHub & GitLab</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-[#96031A]/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-[#96031A]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Jira Integration</h3>
              <p className="text-slate-600 text-sm">Link commits to Jira cards effortlessly</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-[#96031A]/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-[#96031A]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Daily Reports</h3>
              <p className="text-slate-600 text-sm">Automated daily summaries of your work</p>
            </div>

            <div className="text-center">
              <div className="w-14 h-14 bg-[#96031A]/10 rounded-xl flex items-center justify-center mx-auto mb-4">
                <svg className="w-7 h-7 text-[#96031A]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <h3 className="font-semibold text-slate-900 mb-2">Background Sync</h3>
              <p className="text-slate-600 text-sm">Data stays fresh with automatic background updates</p>
            </div>
          </div>
        </div>
      </section>

      {/* How It Works Section */}
      <section className="py-20 bg-[#FBFFFE]">
        <div className="max-w-7xl mx-auto px-4">
          <h2 className="text-3xl font-bold text-center text-slate-900 mb-12">
            How It Works
          </h2>
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="w-12 h-12 bg-[#1B1B1E] text-[#FBFFFE] rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                1
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Connect Your Accounts</h3>
              <p className="text-slate-600">Link your GitHub, GitLab, and Jira accounts with secure tokens</p>
            </div>

            <div className="text-center">
              <div className="w-12 h-12 bg-[#1B1B1E] text-[#FBFFFE] rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                2
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Add Repositories</h3>
              <p className="text-slate-600">Select which repositories you want to track</p>
            </div>

            <div className="text-center">
              <div className="w-12 h-12 bg-[#1B1B1E] text-[#FBFFFE] rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">
                3
              </div>
              <h3 className="text-lg font-semibold text-slate-900 mb-2">Get Daily Reports</h3>
              <p className="text-slate-600">Receive automated daily reports of your development work</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 bg-[#1B1B1E]">
        <div className="max-w-7xl mx-auto px-4 text-center">
          <h2 className="text-3xl font-bold text-white mb-4">
            Start Tracking Your Development Work Today
          </h2>
          <p className="text-slate-300 mb-8 max-w-xl mx-auto">
            Join developers who use PDT to stay organized and track their daily progress effortlessly.
          </p>
          <Link
            to="/register"
            className="inline-block px-8 py-3 bg-[#96031A] text-[#FBFFFE] font-semibold rounded-lg hover:bg-[#96031A]/80 transition-colors"
          >
            Sign Up Free
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 bg-[#1B1B1E] text-[#FBFFFE]">
        <div className="max-w-7xl mx-auto px-4 text-center">
          <p>&copy; 2026 PDT - Personal Development Tracker. All rights reserved.</p>
        </div>
      </footer>
    </div>
  )
}
