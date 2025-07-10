import { useState } from 'react'
import { Clock } from 'lucide-react'
import { TaskInput } from './components/TaskInput'
import { TaskResult } from './components/TaskResult'
import { Header } from './components/Header'
import { ActivityLog } from './components/ActivityLog'

export interface Task {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'error'
  result?: string
  error?: string
  timestamp: Date
  subtasks?: Task[]
  orchestratorId?: string
}

function App() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [currentTask, setCurrentTask] = useState<Task | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const submitTask = async (description: string) => {
    const newTask: Task = {
      id: Date.now().toString(),
      description,
      status: 'pending',
      timestamp: new Date()
    }

    setTasks(prev => [newTask, ...prev])
    setCurrentTask(newTask)
    setIsLoading(true)

    try {
      // Call orchestrator API to submit task
      const response = await fetch('http://localhost:8080/api/task', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ task: description }),
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const submitResult = await response.json()
      const orchestratorTaskId = submitResult.id

      // Update task status to running and store orchestrator ID
      const runningTask = { 
        ...newTask, 
        status: 'running' as const,
        orchestratorId: orchestratorTaskId
      }
      setTasks(prev => prev.map(t => t.id === newTask.id ? runningTask : t))
      setCurrentTask(runningTask)

      // Start polling for task completion
      const pollForResult = async () => {
        while (true) {
          try {
            const statusResponse = await fetch(`http://localhost:8080/api/task/${orchestratorTaskId}`)
            if (!statusResponse.ok) {
              throw new Error(`Failed to get task status: ${statusResponse.status}`)
            }

            const taskStatus = await statusResponse.json()
            
            if (taskStatus.status === 'completed') {
              const completedTask = { 
                ...runningTask, 
                status: 'completed' as const, 
                result: taskStatus.result 
              }
              setTasks(prev => prev.map(t => t.id === newTask.id ? completedTask : t))
              setCurrentTask(completedTask)
              break
            } else if (taskStatus.status === 'error') {
              const errorTask = { 
                ...runningTask, 
                status: 'error' as const, 
                error: taskStatus.error || 'Task failed' 
              }
              setTasks(prev => prev.map(t => t.id === newTask.id ? errorTask : t))
              setCurrentTask(errorTask)
              break
            }
            
            // Wait 2 seconds before polling again
            await new Promise(resolve => setTimeout(resolve, 2000))
          } catch (pollError) {
            console.error('Polling error:', pollError)
            // Continue polling unless it's a critical error
            await new Promise(resolve => setTimeout(resolve, 5000))
          }
        }
      }

      // Start polling in background
      pollForResult()

    } catch (error) {
      const errorTask = { 
        ...newTask, 
        status: 'error' as const, 
        error: error instanceof Error ? error.message : 'Unknown error occurred' 
      }
      setTasks(prev => prev.map(t => t.id === newTask.id ? errorTask : t))
      setCurrentTask(errorTask)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen animated-bg">
      <Header />
      
      <main className="container mx-auto px-8 py-12 max-w-6xl">
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-12">
          {/* Left Column - Task Input - Takes 3/5 of width */}
          <div className="lg:col-span-3 space-y-10">
            <TaskInput onSubmit={submitTask} isLoading={isLoading} />
            
            {/* Task History */}
            <div className="card-3d p-10 relative">
              <div className="flex items-center space-x-4 mb-8">
                <div className="icon-bg">
                  <div className="character-bubble p-4">
                    <Clock className="w-8 h-8 text-white" />
                  </div>
                </div>
                <h2 className="text-3xl font-bold title-gradient">Mission History</h2>
              </div>
              <div className="space-y-6 max-h-96 overflow-y-auto custom-scrollbar max-w-2xl mx-auto">
                {tasks.length === 0 ? (
                  <div className="text-center py-16">
                    <div className="icon-bg mx-auto mb-6">
                      <div className="character-bubble p-5 flex items-center justify-center">
                        <Clock className="w-8 h-8 text-white" />
                      </div>
                    </div>
                    <p className="text-gray-600 text-xl font-medium mb-3">No missions submitted yet</p>
                    <p className="text-gray-400 text-lg">Your AI adventures will appear here</p>
                  </div>
                ) : (
                  tasks.map((task) => (
                    <div 
                      key={task.id} 
                      className={`activity-item cursor-pointer p-6 ${
                        task.status === 'completed' ? 'activity-success' :
                        task.status === 'error' ? 'activity-error' :
                        task.status === 'running' ? 'activity-info' :
                        'activity-warning'
                      }`}
                      onClick={() => setCurrentTask(task)}
                      /* Animation delay removed */
                    >
                      <div className="flex items-center justify-between mb-3">
                        <p className="font-bold text-gray-800 text-xl truncate flex-1 mr-6">
                          {task.description}
                        </p>
                        <div className={`px-4 py-2 rounded-full text-sm font-bold ${
                          task.status === 'completed' ? 'status-success' :
                          task.status === 'error' ? 'status-error' :
                          task.status === 'running' ? 'status-running' :
                          'bg-gray-200 text-gray-800'
                        }`}>
                          {task.status.toUpperCase()}
                        </div>
                      </div>
                      <p className="text-base text-gray-600 font-medium">
                        {task.timestamp.toLocaleString()}
                      </p>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Right Column - Results - Takes 2/5 of width */}
          <div className="lg:col-span-2 space-y-10">
            <TaskResult task={currentTask} />
            <ActivityLog />
          </div>
        </div>
      </main>
    </div>
  )
}

export default App
