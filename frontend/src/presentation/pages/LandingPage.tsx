import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

export function LandingPage() {
  const features = [
    {
      title: 'Automatic Sync',
      description: 'Commits synced automatically from GitHub & GitLab',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
        </svg>
      )
    },
    {
      title: 'Jira Integration',
      description: 'Link commits to Jira cards effortlessly',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
      )
    },
    {
      title: 'Daily Reports',
      description: 'Automated daily summaries of your work',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      )
    },
    {
      title: 'Background Sync',
      description: 'Data stays fresh with automatic updates',
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      )
    }
  ]

  const integrations = [
    { name: 'GitHub', color: 'bg-[#F8C630]' },
    { name: 'GitLab', color: 'bg-[#FC6D26]' },
    { name: 'Jira', color: 'bg-[#0052CC]' }
  ]

  return (
    <div className="min-h-screen">
      {/* Navigation */}
      <header className="fixed top-0 left-0 right-0 z-50 bg-[#F8C630]/95 backdrop-blur-sm">
        <div className="w-full px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <span className="text-2xl font-bold text-[#1B1B1E]">PDT</span>
            </div>
            <nav className="flex items-center gap-6">
              <Link
                to="/login"
                className="text-[#1B1B1E] hover:text-[#96031A] font-medium transition-colors"
              >
                Login
              </Link>
              <Button
                asChild
                className="bg-[#1B1B1E] hover:bg-[#F8C630] hover:text-[#1B1B1E] text-[#F8C630]"
              >
                <Link to="/register">Get Started</Link>
              </Button>
            </nav>
          </div>
        </div>
      </header>

      {/* Hero Section - Yellow with diagonal skew */}
      <section className="relative pt-32 pb-24 px-4 bg-[#F8C630]">
        {/* Diagonal skew at bottom */}
        <div className="absolute bottom-0 left-0 right-0 h-16 bg-[#F8C630]" style={{ transform: 'skewY(-3deg) translateY(50%)' }} />

        <div className="w-full max-w-6xl mx-auto text-center relative z-10">
          <h1 className="text-5xl md:text-7xl font-bold text-[#1B1B1E] mb-6">
            Your Personal
            <br />
            <span className="text-[#1B1B1E]">Development Tracker</span>
          </h1>
          <p className="text-xl md:text-2xl text-[#1B1B1E] mb-10 max-w-2xl mx-auto">
            Automatically track commits across GitHub & GitLab, link Jira cards,
            and generate daily reports.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Button
              asChild
              size="lg"
              className="bg-[#1B1B1E] hover:bg-[#F8C630] hover:text-[#1B1B1E] text-[#F8C630] text-lg px-8"
            >
              <Link to="/register">Get Started Free</Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="lg"
              className="border-2 border-[#1B1B1E] text-[#1B1B1E] hover:bg-[#1B1B1E] hover:text-[#F8C630] text-lg px-8"
            >
              <Link to="/login">View Demo</Link>
            </Button>
          </div>
        </div>
      </section>

      {/* Features Section - Dark background */}
      <section className="py-20 px-4 bg-[#1B1B1E]">
        <div className="w-full max-w-6xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold text-center text-[#FBFFFE] mb-12">
            Everything You Need
          </h2>
          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {features.map((feature, index) => (
              <Card
                key={index}
                className="border border-[#FBFFFE]/20 bg-transparent hover:border-[#F8C630]/50 transition-colors"
              >
                <CardHeader className="text-center pb-2">
                  <div className="w-14 h-14 bg-transparent border-2 border-[#F8C630] rounded-xl flex items-center justify-center mx-auto mb-4 text-[#F8C630]">
                    {feature.icon}
                  </div>
                  <h3 className="text-lg font-bold text-[#FBFFFE]">{feature.title}</h3>
                </CardHeader>
                <CardContent className="text-center pt-0">
                  <p className="text-[#FBFFFE]/70 text-sm">{feature.description}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Integrations Section - Dark background */}
      <section className="py-20 px-4 bg-[#1B1B1E]">
        <div className="w-full max-w-6xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-[#FBFFFE] mb-4">
            Seamless Integrations
          </h2>
          <p className="text-[#FBFFFE]/60 mb-12">
            Connect your favorite development tools
          </p>
          <div className="flex justify-center gap-8">
            {integrations.map((integration) => (
              <div key={integration.name} className="flex flex-col items-center gap-3">
                <div
                  className={`w-24 h-24 ${integration.color} rounded-2xl flex items-center justify-center`}
                >
                  <span className="text-[#1B1B1E] font-bold text-2xl">{integration.name[0]}</span>
                </div>
                <span className="text-[#FBFFFE]/70 font-medium">{integration.name}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works Section - Dark background */}
      <section className="py-20 px-4 bg-[#1B1B1E]">
        <div className="w-full max-w-6xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-[#FBFFFE] mb-4">
            How It Works
          </h2>
          <p className="text-[#FBFFFE]/60 mb-12">
            Get started in three simple steps
          </p>
          <div className="grid md:grid-cols-3 gap-8">
            {[
              { step: '01', title: 'Connect', desc: 'Link your GitHub, GitLab, and Jira accounts' },
              { step: '02', title: 'Track', desc: 'Add repositories you want to monitor' },
              { step: '03', title: 'Report', desc: 'Receive daily reports automatically' }
            ].map((item, i) => (
              <div key={i} className="text-center">
                <div className="text-6xl font-bold text-[#F8C630]/30 mb-4">{item.step}</div>
                <h3 className="text-xl font-bold text-[#FBFFFE] mb-2">{item.title}</h3>
                <p className="text-[#FBFFFE]/60">{item.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section - Yellow background */}
      <section className="py-20 px-4 bg-[#F8C630]">
        <div className="w-full max-w-4xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-[#1B1B1E] mb-4">
            Start Tracking Today
          </h2>
          <p className="text-xl text-[#1B1B1E] mb-8">
            Join developers who use PDT to stay organized.
          </p>
          <Button
            asChild
            size="lg"
            className="bg-[#1B1B1E] hover:bg-[#96031A] text-[#F8C630] text-lg px-8"
          >
            <Link to="/register">Sign Up Free</Link>
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 px-4 bg-[#1B1B1E] border-t border-[#F8C630]">
        <div className="w-full max-w-6xl mx-auto text-center">
          <p className="text-[#FBFFFE]/60">
            &copy; 2026 PDT - Personal Development Tracker
          </p>
        </div>
      </footer>
    </div>
  )
}
