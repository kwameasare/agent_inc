import { useState } from 'react'
import { Send, Loader2, Lightbulb } from 'lucide-react'

interface TaskInputProps {
  onSubmit: (task: string) => void
  isLoading: boolean
}

const exampleTasks = [
  "Analyze the market trends for electric vehicles in 2024",
  "Create a comprehensive marketing strategy for a new SaaS product",
  "Design a database schema for an e-commerce platform",
  "Write a technical blog post about microservices architecture",
  "Research and summarize the latest developments in quantum computing"
]

export function TaskInput({ onSubmit, isLoading }: TaskInputProps) {
  const [task, setTask] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (task.trim() && !isLoading) {
      onSubmit(task.trim())
      setTask('')
    }
  }

  const handleExampleClick = (exampleTask: string) => {
    setTask(exampleTask)
  }

  return (
    <div className="card-3d p-10">
      <div className="flex items-center space-x-6 mb-10">
        <div className="icon-bg">
          <div className="character-bubble p-4">
            <Send className="w-8 h-8 text-white" />
          </div>
        </div>
        <h2 className="text-4xl font-bold title-gradient">Submit New Mission</h2>
      </div>
      
      <form onSubmit={handleSubmit} className="space-y-10 max-w-3xl mx-auto">
        <div>
          <label htmlFor="task" className="block text-xl font-bold subtitle-gradient mb-6">
            Mission Description
          </label>
          <textarea
            id="task"
            value={task}
            onChange={(e) => setTask(e.target.value)}
            placeholder="Enter a complex mission that requires multiple AI agents to collaborate..."
            className="input-modern w-full h-48 resize-none text-lg"
            disabled={isLoading}
          />
        </div>
        
        <div className="text-center">
          <button
            type="submit"
            disabled={!task.trim() || isLoading}
            className="btn-glow inline-flex items-center justify-center space-x-4 disabled:opacity-50 disabled:cursor-not-allowed py-6 px-10"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-7 h-7 animate-spin" />
                <span className="text-2xl">AI Agents Processing...</span>
              </>
            ) : (
              <>
                <Send className="w-7 h-7" />
                <span className="text-2xl">Launch AI Mission</span>
              </>
            )}
          </button>
        </div>
      </form>
      
      {/* Example Tasks */}
      <div className="mt-12 pt-10 border-t border-gray-200 max-w-3xl mx-auto">
        <div className="flex items-center space-x-4 mb-8">
          <div className="icon-bg">
            <div className="character-bubble p-3">
              <Lightbulb className="w-7 h-7 text-white" />
            </div>
          </div>
          <h3 className="text-2xl font-bold subtitle-gradient">Example Missions</h3>
        </div>
        <div className="grid gap-4">
          {exampleTasks.map((example, index) => (
            <button
              key={index}
              onClick={() => handleExampleClick(example)}
              className="activity-item text-left hover:scale-[1.02] transition-all duration-300 border-l-4 activity-info p-6"
              disabled={isLoading}
              /* Animation delay removed */
            >
              <span className="font-semibold text-gray-700 text-lg">{example}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
