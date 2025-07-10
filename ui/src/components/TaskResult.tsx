import { CheckCircle, XCircle, Clock, Loader2, FileText, AlertTriangle, Users, CheckCircle2, Pause, Eye } from 'lucide-react'
import { useState } from 'react'
import type { Task } from '../App'
import { ExpertResultsModal } from './ExpertResultsModal'

interface TaskResultProps {
  task: Task | null
  onApprovePhase?: (taskId: string, phaseId: string, approved: boolean, feedback?: string) => Promise<any>
}

// NEW component for displaying a single expert with expandable details
const ExpertDetails = ({ expert }: { expert: any }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const getExpertStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="w-5 h-5 text-green-600" />
      case 'running':
        return <Loader2 className="w-5 h-5 text-blue-600 animate-spin" />
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-600" />
      default:
        return <Clock className="w-5 h-5 text-gray-500" />
    }
  }

  return (
    <div className="bg-white bg-opacity-50 rounded-lg p-3 my-2">
      <div
        className="flex items-center justify-between cursor-pointer"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center space-x-3">
          {getExpertStatusIcon(expert.status)}
          <div>
            <p className="font-semibold text-gray-800">{expert.role}</p>
            <p className="text-sm text-gray-600">{expert.expertise}</p>
          </div>
        </div>
        <div className={`px-2 py-1 rounded text-xs font-bold ${
          expert.status === 'completed' ? 'bg-green-100 text-green-800' :
          expert.status === 'running' ? 'bg-blue-100 text-blue-800' :
          expert.status === 'failed' ? 'bg-red-100 text-red-800' :
          'bg-gray-100 text-gray-600'
        }`}>
          {expert.status.toUpperCase()}
        </div>
      </div>
      {isExpanded && expert.status === 'completed' && (
        <div className="mt-3 pt-3 border-t border-gray-200">
          <h6 className="font-bold text-gray-800 text-sm mb-1">Expert's Report:</h6>
          <pre className="text-xs text-gray-700 whitespace-pre-wrap font-mono bg-gray-50 p-2 rounded">
            {expert.result || "No result was provided by the agent."}
          </pre>
        </div>
      )}
    </div>
  );
};

export function TaskResult({ task, onApprovePhase }: TaskResultProps) {
  const [phaseApprovalFeedback, setPhaseApprovalFeedback] = useState<{ [key: string]: string }>({})
  const [expertResultsModal, setExpertResultsModal] = useState<{
    isOpen: boolean
    taskId: string
    phaseId: string
    phaseName: string
  }>({
    isOpen: false,
    taskId: '',
    phaseId: '',
    phaseName: ''
  })

  const openExpertResults = (taskId: string, phaseId: string, phaseName: string) => {
    setExpertResultsModal({
      isOpen: true,
      taskId,
      phaseId,
      phaseName
    })
  }

  const closeExpertResults = () => {
    setExpertResultsModal({
      isOpen: false,
      taskId: '',
      phaseId: '',
      phaseName: ''
    })
  }

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
    if (task.phases && task.phases.length > 0) {
      const awaitingApprovalPhase = task.phases.find(p => p.status === 'awaiting_approval')
      if (awaitingApprovalPhase) {
        return 'Awaiting Phase Approval'
      }
    }
    
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

  const getPhaseStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-6 h-6 text-green-600" />
      case 'approved':
        return <CheckCircle className="w-6 h-6 text-green-600" />
      case 'running':
        return <Loader2 className="w-6 h-6 text-blue-600 animate-spin" />
      case 'awaiting_approval':
        return <Pause className="w-6 h-6 text-yellow-600" />
      case 'rejected':
        return <XCircle className="w-6 h-6 text-red-600" />
      default:
        return <Clock className="w-6 h-6 text-gray-500" />
    }
  }

  const handlePhaseApproval = async (phaseId: string, approved: boolean) => {
    if (!task.orchestratorId || !onApprovePhase) return
    
    try {
      const feedback = phaseApprovalFeedback[phaseId] || ''
      await onApprovePhase(task.orchestratorId, phaseId, approved, feedback)
      
      // Clear feedback after approval
      setPhaseApprovalFeedback(prev => {
        const updated = { ...prev }
        delete updated[phaseId]
        return updated
      })
    } catch (error) {
      console.error('Failed to approve phase:', error)
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

        {/* Phases (if any) */}
        {task.phases && task.phases.length > 0 && (
          <div>
            <h3 className="text-2xl font-bold subtitle-gradient mb-6 flex items-center space-x-4">
              <div className="icon-bg">
                <div className="character-bubble p-3">
                  <Users className="w-7 h-7 text-white" />
                </div>
              </div>
              <span>Project Phases</span>
            </h3>
            <div className="space-y-6">
              {task.phases.map((phase, phaseIndex) => (
                <div 
                  key={phase.id}
                  className={`activity-item p-6 ${
                    phase.status === 'completed' || phase.status === 'approved' ? 'activity-success' :
                    phase.status === 'rejected' ? 'activity-error' :
                    phase.status === 'running' ? 'activity-info' :
                    phase.status === 'awaiting_approval' ? 'activity-warning' :
                    'bg-gray-100'
                  }`}
                >
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center space-x-3">
                      {getPhaseStatusIcon(phase.status)}
                      <h4 className="text-xl font-bold text-gray-800">
                        Phase {phaseIndex + 1}: {phase.name}
                      </h4>
                    </div>
                    <div className="flex items-center space-x-3">
                      {/* View Details Button */}
                      {phase.status === 'completed' || phase.status === 'approved' ? (
                        <button
                          onClick={() => openExpertResults(task.orchestratorId || task.id, phase.id, phase.name)}
                          className="flex items-center space-x-2 px-3 py-1 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
                        >
                          <Eye className="w-4 h-4" />
                          <span>View Details</span>
                        </button>
                      ) : null}
                      <div className={`px-3 py-1 rounded-full text-sm font-bold ${
                        phase.status === 'completed' || phase.status === 'approved' ? 'status-success' :
                        phase.status === 'rejected' ? 'status-error' :
                        phase.status === 'running' ? 'status-running' :
                        phase.status === 'awaiting_approval' ? 'bg-yellow-200 text-yellow-800' :
                        'bg-gray-200 text-gray-800'
                      }`}>
                        {phase.status.toUpperCase()}
                      </div>
                    </div>
                  </div>
                  
                  <p className="text-gray-700 font-medium mb-4">{phase.description}</p>
                  
                  {/* Domain Experts */}
                  {phase.experts && phase.experts.length > 0 && (
                    <div className="ml-4 space-y-3">
                      <h5 className="text-lg font-bold text-gray-700 mb-3">Domain Experts:</h5>
                      {phase.experts.map((expert, expertIndex) => (
                        <ExpertDetails key={expertIndex} expert={expert} />
                      ))}
                    </div>
                  )}
                  
                  {/* Phase Results */}
                  {phase.results && Object.keys(phase.results).length > 0 && (
                    <div className="mt-4 p-4 bg-white bg-opacity-30 rounded-lg">
                      <h5 className="text-lg font-bold text-gray-700 mb-2">Phase Results:</h5>
                      {Object.entries(phase.results).map(([expertRole, result]) => (
                        <div key={expertRole} className="mb-3">
                          <p className="font-semibold text-gray-800">{expertRole}:</p>
                          <p className="text-gray-700 text-sm mt-1 pl-4 border-l-2 border-gray-300">
                            {result.length > 200 ? `${result.substring(0, 200)}...` : result}
                          </p>
                        </div>
                      ))}
                    </div>
                  )}
                  
                  {/* Phase Approval UI */}
                  {phase.status === 'awaiting_approval' && onApprovePhase && task.orchestratorId && (
                    <div className="mt-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                      <h5 className="text-lg font-bold text-yellow-800 mb-3">
                        Phase Approval Required
                      </h5>
                      <div className="space-y-4">
                        <textarea
                          className="w-full p-3 border border-yellow-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-yellow-500"
                          placeholder="Optional feedback for this phase..."
                          value={phaseApprovalFeedback[phase.id] || ''}
                          onChange={(e) => setPhaseApprovalFeedback(prev => ({
                            ...prev,
                            [phase.id]: e.target.value
                          }))}
                          rows={3}
                        />
                        <div className="flex space-x-4">
                          <button
                            onClick={() => handlePhaseApproval(phase.id, true)}
                            className="flex items-center space-x-2 px-6 py-3 bg-green-600 text-white font-bold rounded-lg hover:bg-green-700 transition-colors"
                          >
                            <CheckCircle className="w-5 h-5" />
                            <span>Approve Phase</span>
                          </button>
                          <button
                            onClick={() => handlePhaseApproval(phase.id, false)}
                            className="flex items-center space-x-2 px-6 py-3 bg-red-600 text-white font-bold rounded-lg hover:bg-red-700 transition-colors"
                          >
                            <XCircle className="w-5 h-5" />
                            <span>Reject Phase</span>
                          </button>
                        </div>
                      </div>
                    </div>
                  )}
                  
                  {/* User Feedback Display */}
                  {phase.userFeedback && (
                    <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
                      <h5 className="font-semibold text-blue-800 mb-2">User Feedback:</h5>
                      <p className="text-blue-700">{phase.userFeedback}</p>
                    </div>
                  )}
                </div>
              ))}
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
      
      {/* Expert Results Modal */}
      <ExpertResultsModal
        taskId={expertResultsModal.taskId}
        phaseId={expertResultsModal.phaseId}
        phaseName={expertResultsModal.phaseName}
        isOpen={expertResultsModal.isOpen}
        onClose={closeExpertResults}
      />
    </div>
  )
}
