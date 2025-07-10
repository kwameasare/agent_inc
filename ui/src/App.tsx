import { useState, useEffect, useRef } from 'react'
import { Clock } from 'lucide-react'
import { TaskInput } from './components/TaskInput'
import { TaskResult } from './components/TaskResult'
import { Header } from './components/Header'
import { ActivityLog } from './components/ActivityLog'

export interface DomainExpert {
  role: string
  expertise: string
  persona: string
  task: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  result?: string
}

export interface ProjectPhase {
  id: string
  name: string
  description: string
  status: 'pending' | 'approved' | 'running' | 'completed' | 'rejected' | 'awaiting_approval'
  experts: DomainExpert[]
  results?: { [key: string]: string }
  startTime?: string
  endTime?: string
  approved: boolean
  userFeedback?: string
}

export interface Task {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'error' | 'failed'
  result?: string
  error?: string
  timestamp: Date
  subtasks?: Task[]
  orchestratorId?: string
  phases?: ProjectPhase[]
  currentPhase?: number
  requiresUserApproval?: boolean
}

function App() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [currentTask, setCurrentTask] = useState<Task | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  // Load tasks from localStorage on component mount
  useEffect(() => {
    const savedTasks = localStorage.getItem('agent_inc_tasks')
    const savedCurrentTaskId = localStorage.getItem('agent_inc_current_task')
    
    if (savedTasks) {
      try {
        const parsedTasks = JSON.parse(savedTasks).map((task: any) => ({
          ...task,
          timestamp: new Date(task.timestamp)
        }))
        setTasks(parsedTasks)
        
        if (savedCurrentTaskId) {
          const currentTask = parsedTasks.find((task: Task) => task.id === savedCurrentTaskId)
          if (currentTask) {
            setCurrentTask(currentTask)
            // Fetch latest status for the current task
            fetchTaskStatus(currentTask.orchestratorId || currentTask.id)
          }
        }
      } catch (error) {
        console.error('Failed to load tasks from localStorage:', error)
      }
    }
  }, [])

  // Save tasks to localStorage whenever tasks change
  useEffect(() => {
    localStorage.setItem('agent_inc_tasks', JSON.stringify(tasks))
  }, [tasks])

  // Save current task ID to localStorage whenever it changes
  useEffect(() => {
    if (currentTask) {
      localStorage.setItem('agent_inc_current_task', currentTask.id)
    } else {
      localStorage.removeItem('agent_inc_current_task')
    }
  }, [currentTask])

  // WebSocket connection and message handling
  useEffect(() => {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${wsProtocol}//${window.location.host}/ws`
    
    const connect = () => {
      wsRef.current = new WebSocket(wsUrl)
      
      wsRef.current.onopen = () => {
        console.log('WebSocket connected')
        setIsConnected(true)
      }
      
      wsRef.current.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)
        // Attempt to reconnect after 3 seconds
        setTimeout(connect, 3000)
      }
      
      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
      
      wsRef.current.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)
          handleWebSocketMessage(message)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }
    }
    
    connect()
    
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  // SSE connection for real-time task updates
  useEffect(() => {
    if (currentTask?.status === 'running' && currentTask.orchestratorId) {
      const eventSource = new EventSource(`http://localhost:8080/api/task/${currentTask.orchestratorId}/events`)
      eventSourceRef.current = eventSource
      
      eventSource.onmessage = (event) => {
        try {
          const updatedTaskData = JSON.parse(event.data)
          const taskToUpdate: Task = {
            ...updatedTaskData,
            timestamp: new Date(updatedTaskData.started || updatedTaskData.createdAt),
          }
          
          setCurrentTask(taskToUpdate)
          setTasks(prev => prev.map(t => t.orchestratorId === taskToUpdate.id ? taskToUpdate : t))

          if (taskToUpdate.status === 'completed' || taskToUpdate.status === 'failed' || taskToUpdate.status === 'error') {
            eventSource.close()
          }
        } catch (error) {
          console.error('Failed to parse SSE message:', error)
        }
      }

      eventSource.onerror = (error) => {
        console.error('SSE error:', error)
        eventSource.close()
        // Fall back to periodic polling if SSE fails
        console.log('Falling back to WebSocket updates')
      }

      return () => {
        eventSource.close()
      }
    }
  }, [currentTask?.orchestratorId, currentTask?.status])

  const handleWebSocketMessage = (message: any) => {
    const { type, taskId } = message
    
    switch (type) {
      case 'task_created':
      case 'task_status_updated':
      case 'plan_generated':
      case 'phase_started':
      case 'phase_completed':
      case 'phase_awaiting_approval':
      case 'phase_approved':
      case 'phase_rejected':
      case 'expert_started':
      case 'expert_completed':
      case 'expert_failed':
      case 'task_completed':
        // Update the specific task in our state
        updateTaskFromWebSocket(taskId)
        break
      default:
        console.log('Unknown WebSocket message type:', type)
    }
  }

  const updateTaskFromWebSocket = async (taskId: string) => {
    // Fetch the latest task status from the API
    try {
      const response = await fetch(`http://localhost:8080/api/task/${taskId}`)
      if (response.ok) {
        const updatedTask = await response.json()
        
        setTasks(prev => prev.map(task => {
          if (task.orchestratorId === taskId || task.id === taskId) {
            const updated = {
              ...task,
              status: updatedTask.status,
              result: updatedTask.result,
              error: updatedTask.error,
              phases: updatedTask.phases,
              currentPhase: updatedTask.currentPhase,
              requiresUserApproval: updatedTask.requiresUserApproval
            }
            
            // Update current task if it's the one being updated
            if (currentTask && (currentTask.orchestratorId === taskId || currentTask.id === taskId)) {
              setCurrentTask(updated)
            }
            
            return updated
          }
          return task
        }))
      }
    } catch (error) {
      console.error('Failed to fetch updated task status:', error)
    }
  }

  const fetchTaskStatus = async (taskId: string) => {
    try {
      const response = await fetch(`http://localhost:8080/api/task/${taskId}`)
      if (response.ok) {
        const taskStatus = await response.json()
        
        setTasks(prev => prev.map(task => {
          if (task.orchestratorId === taskId || task.id === taskId) {
            const updated = {
              ...task,
              status: taskStatus.status,
              result: taskStatus.result,
              error: taskStatus.error,
              phases: taskStatus.phases,
              currentPhase: taskStatus.currentPhase,
              requiresUserApproval: taskStatus.requiresUserApproval
            }
            
            if (currentTask && (currentTask.orchestratorId === taskId || currentTask.id === taskId)) {
              setCurrentTask(updated)
            }
            
            return updated
          }
          return task
        }))
      }
    } catch (error) {
      console.error('Failed to fetch task status:', error)
    }
  }

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

      console.log(`Task submitted successfully. Orchestrator ID: ${orchestratorTaskId}`)
      console.log('SSE and WebSocket will handle real-time updates')

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

  const approvePhase = async (taskId: string, phaseId: string, approved: boolean, feedback?: string) => {
    try {
      const response = await fetch('http://localhost:8080/api/phases/approve', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          taskId,
          phaseId,
          approved,
          userFeedback: feedback || ''
        }),
      })

      if (!response.ok) {
        throw new Error(`Failed to approve phase: ${response.status}`)
      }

      const result = await response.json()
      
      // Update the current task with the approved phase
      if (currentTask && currentTask.orchestratorId === taskId) {
        setTasks(prev => prev.map(task => {
          if (task.orchestratorId === taskId) {
            const updatedPhases = task.phases?.map(phase => 
              phase.id === phaseId 
                ? { ...phase, approved, userFeedback: feedback, status: (approved ? 'approved' : 'rejected') as ProjectPhase['status'] }
                : phase
            )
            return { ...task, phases: updatedPhases }
          }
          return task
        }))
        
        setCurrentTask(prev => {
          if (prev && prev.orchestratorId === taskId) {
            const updatedPhases = prev.phases?.map(phase => 
              phase.id === phaseId 
                ? { ...phase, approved, userFeedback: feedback, status: (approved ? 'approved' : 'rejected') as ProjectPhase['status'] }
                : phase
            )
            return { ...prev, phases: updatedPhases }
          }
          return prev
        })
      }
      
      return result
    } catch (error) {
      console.error('Phase approval error:', error)
      throw error
    }
  }

  return (
    <div className="min-h-screen animated-bg">
      <Header />
      
      {/* WebSocket Status Indicator */}
      <div className={`fixed top-4 right-4 px-3 py-1 rounded-full text-sm font-medium z-50 ${
        isConnected 
          ? 'bg-green-500 text-white' 
          : 'bg-red-500 text-white animate-pulse'
      }`}>
        {isConnected ? '🟢 Live' : '🔴 Disconnected'}
      </div>
      
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
                      } ${currentTask?.id === task.id ? 'ring-2 ring-blue-400' : ''}`}
                      onClick={() => setCurrentTask(task)}
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
                      <div className="flex items-center justify-between">
                        <p className="text-base text-gray-600 font-medium">
                          {task.timestamp.toLocaleString()}
                        </p>
                        {task.phases && task.phases.length > 0 && (
                          <p className="text-sm text-gray-500">
                            Phase {(task.currentPhase || 0) + 1} of {task.phases.length}
                          </p>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Right Column - Results - Takes 2/5 of width */}
          <div className="lg:col-span-2 space-y-10">
            <TaskResult task={currentTask} onApprovePhase={approvePhase} />
            <ActivityLog />
          </div>
        </div>
      </main>
    </div>
  )
}

export default App
