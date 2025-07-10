import { CheckCircle, XCircle, Clock, Loader2, FileText, AlertTriangle } from 'lucide-react'
import type { Task } from '../App'

interface TaskResultProps {
  task: Task | null
}

export function TaskResult({ task }: TaskResultProps) {
  if (!task) {
    return (
      <div className="card-3d p-10 text-center relative">
        <div className="py-20">
          <div className="icon-bg mx-auto mb-8">
            <div className="character-bubble p-6 flex items-center justify-center">
              <FileText className="w-10 h-10 text-white" />
            </div>
          </div>
          <h3 className="text-4xl font-bold title-gradient mb-4">No Mission Selected</h3>
          <p className="subtitle-gradient text-xl">Submit a mission to see AI magic happen here</p>
        </div>
      </div>
    )
  }

  const getStatusIcon = () => {
    switch (task.status) {
      case 'completed':
        return <CheckCircle className="w-8 h-8 text-white" />
      case 'error':
        return <XCircle className="w-8 h-8 text-white" />
      case 'running':
        return <Loader2 className="w-8 h-8 text-white animate-spin" />
      default:
        return <Clock className="w-8 h-8 text-white" />
    }
  }

  const getStatusText = () => {
    switch (task.status) {
      case 'completed':
        return 'Mission Accomplished!'
      case 'error':
        return 'Mission Failed'
      case 'running':
        return 'AI Agents Working...'
      default:
        return 'Mission Pending'
    }
  }

  const getStatusStyle = () => {
    switch (task.status) {
      case 'completed':
        return 'status-success'
      case 'error':
        return 'status-error'
      case 'running':
        return 'status-running'
      default:
        return 'bg-gray-200 text-gray-800'
    }
  }

  return (
    <div className="card-3d p-10 relative">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-10 relative">
        <h2 className="text-4xl font-bold title-gradient">Mission Results</h2>
        <div className={`flex items-center space-x-4 px-6 py-4 rounded-xl text-lg font-bold ${getStatusStyle()}`}>
          <div className="icon-bg">
            <div className="character-bubble p-3">
              {getStatusIcon()}
            </div>
          </div>
          <span>{getStatusText()}</span>
        </div>
      </div>
      
      <div className="space-y-10 max-w-lg mx-auto">
        {/* Task Description */}
        <div>
          <h3 className="text-2xl font-bold subtitle-gradient mb-6">Mission Brief</h3>
          <div className="activity-item activity-info p-6">
            <p className="text-gray-800 font-semibold text-xl leading-relaxed">
              {task.description}
            </p>
          </div>
        </div>
        
        {/* Timestamp */}
        <div>
          <h3 className="text-2xl font-bold subtitle-gradient mb-6">Mission Started</h3>
          <div className="activity-item activity-info p-6">
            <p className="text-blue-800 font-bold text-xl">
              {task.timestamp.toLocaleString()}
            </p>
          </div>
        </div>

        {/* Result or Error */}
        {task.status === 'completed' && task.result && (
          <div>
            <h3 className="text-2xl font-bold subtitle-gradient mb-6">Mission Success Report</h3>
            <div className="activity-item activity-success p-6">
              <pre className="text-green-800 font-semibold whitespace-pre-wrap leading-relaxed text-lg">
                {task.result}
              </pre>
            </div>
          </div>
        )}

        {task.status === 'error' && task.error && (
          <div>
            <h3 className="text-2xl font-bold subtitle-gradient mb-6 flex items-center space-x-4">
              <div className="icon-bg">
                <div className="character-bubble p-3">
                  <AlertTriangle className="w-7 h-7 text-white" />
                </div>
              </div>
              <span>Mission Error Report</span>
            </h3>
            <div className="activity-item activity-error p-6">
              <p className="text-red-800 font-semibold text-xl">
                {task.error}
              </p>
            </div>
          </div>
        )}

        {task.status === 'running' && (
          <div>
            <h3 className="text-2xl font-bold subtitle-gradient mb-6">Mission Status</h3>
            <div className="activity-item activity-info p-8">
              <div className="flex items-center space-x-6">
                <div className="icon-bg">
                  <div className="character-bubble p-4">
                    <Loader2 className="w-8 h-8 text-white animate-spin" />
                  </div>
                </div>
                <div>
                  <p className="text-blue-800 font-bold text-2xl">
                    Multi-Agent AI Processing in Progress
                  </p>
                  <p className="text-blue-600 font-semibold text-lg mt-2">
                    Our specialized AI agents are collaborating to complete your mission...
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Subtasks (if any) */}
        {task.subtasks && task.subtasks.length > 0 && (
          <div>
            <h3 className="text-2xl font-bold subtitle-gradient mb-6">Mission Breakdown</h3>
            <div className="space-y-6">
              {task.subtasks.map((subtask, index) => (
                <div 
                  key={index}
                  className="activity-item activity-info p-6"
                  /* Animation delay removed */
                >
                  <div className="flex items-center justify-between">
                    <p className="text-gray-800 font-semibold text-xl">{subtask.description}</p>
                    <div className={`px-4 py-2 rounded-lg text-sm font-bold ${getStatusStyle()}`}>
                      {getStatusText()}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
