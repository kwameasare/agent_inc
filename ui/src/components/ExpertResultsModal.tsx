import { useState } from 'react'
import { ChevronDown, ChevronUp, FileText, User } from 'lucide-react'

interface ExpertResult {
  expertise: string
  task: string
  status: string
  result: string
}

interface ExpertResultsModalProps {
  taskId: string
  phaseId: string
  phaseName: string
  isOpen: boolean
  onClose: () => void
}

export function ExpertResultsModal({ taskId, phaseId, phaseName, isOpen, onClose }: ExpertResultsModalProps) {
  const [expertResults, setExpertResults] = useState<{ [key: string]: ExpertResult }>({})
  const [loading, setLoading] = useState(false)
  const [expandedExperts, setExpandedExperts] = useState<{ [key: string]: boolean }>({})

  const fetchExpertResults = async () => {
    if (loading) return
    
    setLoading(true)
    try {
      const response = await fetch(`http://localhost:8081/api/phase/${taskId}/${phaseId}`)
      if (response.ok) {
        const data = await response.json()
        setExpertResults(data.detailedResults || {})
      } else {
        console.error('Failed to fetch expert results')
      }
    } catch (error) {
      console.error('Error fetching expert results:', error)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpertExpanded = (expertRole: string) => {
    setExpandedExperts(prev => ({
      ...prev,
      [expertRole]: !prev[expertRole]
    }))
  }

  // Fetch results when modal opens
  if (isOpen && Object.keys(expertResults).length === 0 && !loading) {
    fetchExpertResults()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
        <div className="bg-gradient-to-r from-blue-600 to-purple-600 text-white p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="bg-white bg-opacity-20 p-2 rounded-lg">
                <FileText className="w-6 h-6" />
              </div>
              <div>
                <h2 className="text-2xl font-bold">{phaseName}</h2>
                <p className="text-blue-100">Expert Results & Deliverables</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="bg-white bg-opacity-20 hover:bg-opacity-30 p-2 rounded-lg transition-colors"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
        
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-120px)]">
          {loading ? (
            <div className="text-center py-12">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading expert results...</p>
            </div>
          ) : Object.keys(expertResults).length === 0 ? (
            <div className="text-center py-12">
              <div className="bg-gray-100 p-4 rounded-lg inline-block mb-4">
                <User className="w-12 h-12 text-gray-400" />
              </div>
              <p className="text-gray-600">No expert results available for this phase.</p>
            </div>
          ) : (
            <div className="space-y-6">
              {Object.entries(expertResults).map(([expertRole, result]) => (
                <div key={expertRole} className="border border-gray-200 rounded-lg overflow-hidden">
                  <div 
                    className="bg-gray-50 p-4 cursor-pointer hover:bg-gray-100 transition-colors"
                    onClick={() => toggleExpertExpanded(expertRole)}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className="bg-blue-100 p-2 rounded-lg">
                          <User className="w-5 h-5 text-blue-600" />
                        </div>
                        <div>
                          <h3 className="text-lg font-bold text-gray-800">{expertRole}</h3>
                          <p className="text-sm text-gray-600">{result.expertise}</p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-3">
                        <div className={`px-3 py-1 rounded-full text-xs font-bold ${
                          result.status === 'completed' ? 'bg-green-100 text-green-800' :
                          result.status === 'running' ? 'bg-blue-100 text-blue-800' :
                          result.status === 'failed' ? 'bg-red-100 text-red-800' :
                          'bg-gray-100 text-gray-600'
                        }`}>
                          {result.status.toUpperCase()}
                        </div>
                        {expandedExperts[expertRole] ? 
                          <ChevronUp className="w-5 h-5 text-gray-500" /> :
                          <ChevronDown className="w-5 h-5 text-gray-500" />
                        }
                      </div>
                    </div>
                  </div>
                  
                  {expandedExperts[expertRole] && (
                    <div className="p-6 bg-white border-t border-gray-200">
                      <div className="space-y-4">
                        <div>
                          <h4 className="text-sm font-bold text-gray-700 mb-2">Task Assignment:</h4>
                          <p className="text-gray-600 bg-gray-50 p-3 rounded-lg">{result.task}</p>
                        </div>
                        
                        {result.result && (
                          <div>
                            <h4 className="text-sm font-bold text-gray-700 mb-2">Deliverable:</h4>
                            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                              <pre className="text-green-800 whitespace-pre-wrap leading-relaxed text-sm">
                                {result.result}
                              </pre>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
