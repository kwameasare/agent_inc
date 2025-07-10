import { Activity, Bot, Users, Clock } from 'lucide-react'

export function ActivityLog() {
  // This would typically come from a real-time API or WebSocket
  const activities = [
    {
      id: '1',
      timestamp: new Date(Date.now() - 30000),
      type: 'agent_spawn',
      message: 'AI Agent successfully deployed and ready for action',
      icon: Bot,
      style: 'activity-success'
    },
    {
      id: '2',
      timestamp: new Date(Date.now() - 45000),
      type: 'task_delegation',
      message: 'Mission delegated to specialized AI sub-agent',
      icon: Users,
      style: 'activity-info'
    },
    {
      id: '3',
      timestamp: new Date(Date.now() - 60000),
      type: 'processing',
      message: 'Advanced AI models processing complex analysis',
      icon: Activity,
      style: 'activity-warning'
    },
    {
      id: '4',
      timestamp: new Date(Date.now() - 120000),
      type: 'system',
      message: 'Multi-Agent Orchestrator initialized and operational',
      icon: Clock,
      style: 'activity-info'
    }
  ]

  return (
    <div className="card-3d p-10 relative">
      <div className="flex items-center space-x-6 mb-10">
        <div className="icon-bg">
          <div className="character-bubble p-4">
            <Activity className="w-8 h-8 text-white" />
          </div>
        </div>
        <h2 className="text-4xl font-bold title-gradient">Live Mission Feed</h2>
      </div>
      
      <div className="space-y-6 max-h-96 overflow-y-auto custom-scrollbar max-w-lg mx-auto">
        {activities.length === 0 ? (
          <div className="text-center py-16">
            <div className="icon-bg mx-auto mb-6">
              <div className="character-bubble p-5 flex items-center justify-center">
                <Activity className="w-8 h-8 text-white" />
              </div>
            </div>
            <p className="subtitle-gradient text-xl font-medium">No recent activity</p>
          </div>
        ) : (
          activities.map((activity) => {
            const IconComponent = activity.icon
            return (
              <div 
                key={activity.id} 
                className={`activity-item ${activity.style} p-6`}
                /* Animation delay removed */
              >
                <div className="flex items-start space-x-6">
                  <div className="icon-bg">
                    <div className="character-bubble p-4">
                      <IconComponent className="w-7 h-7 text-white" />
                    </div>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-gray-800 font-bold text-xl leading-snug">
                      {activity.message}
                    </p>
                    <div className="flex items-center space-x-3 mt-3">
                      <Clock className="w-5 h-5 text-gray-500" />
                      <p className="text-base text-gray-600 font-semibold">
                        {activity.timestamp.toLocaleTimeString()}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>
      
      <div className="mt-10 pt-8 border-t border-gray-200">
        <div className="flex items-center justify-center space-x-4">
          <div className="w-4 h-4 bg-gradient-cyber rounded-full animate-pulse"></div>
          <p className="text-base subtitle-gradient font-bold">
            Real-time AI monitoring • Last updated {new Date().toLocaleTimeString()}
          </p>
        </div>
      </div>
    </div>
  )
}
