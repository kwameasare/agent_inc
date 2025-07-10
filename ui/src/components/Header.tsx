import { Bot, GitBranch } from 'lucide-react'

export function Header() {
  return (
    <header className="bg-white/80 backdrop-blur-lg border-b border-pink-200/30 shadow-sm">
      <div className="container mx-auto px-8 py-12 max-w-7xl">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-8">
            <div className="icon-bg">
              <div className="character-bubble p-5">
                <Bot className="w-12 h-12 text-white" />
              </div>
            </div>
            <div>
              <h1 className="title-gradient mb-3 text-5xl">Agent Orchestrator</h1>
              <p className="subtitle-gradient text-2xl">Multi-Agent AI Task Delegation System</p>
            </div>
          </div>
          
          <div className="flex items-center space-x-6 bg-white/60 backdrop-blur-sm px-6 py-3 rounded-full border border-pink-200/40 shadow-lg">
            <div className="icon-bg">
              <div className="character-bubble p-2">
                <GitBranch className="w-6 h-6 text-white" />
              </div>
            </div>
            <span className="text-gray-700 font-bold text-lg">Hierarchical Processing</span>
          </div>
        </div>
      </div>
    </header>
  )
}
