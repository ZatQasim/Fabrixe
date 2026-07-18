import { useState } from 'react'
import { Rocket, Plus, Play, Trash2, Clock, CheckCircle, XCircle, Loader, CalendarClock } from 'lucide-react'
import { deployment } from '../lib/api'
import { useApi } from '../hooks/useApi'
import { formatDate, formatRelative } from '../lib/format'
import { StatusBadge } from '../components/ui/StatusDot'
import Modal from '../components/ui/Modal'
import { PageLoader } from '../components/ui/Spinner'
import EmptyState from '../components/ui/EmptyState'
import type { Deployment as Dep, ScheduledTask } from '../types'

type DeployForm = { name: string; description: string; deploy_type: 'script' | 'docker' | 'systemd'; config: string }
type TaskForm = { name: string; description: string; command: string; schedule: string; is_active: boolean }

const emptyDeploy: DeployForm = { name: '', description: '', deploy_type: 'script', config: '' }
const emptyTask: TaskForm = { name: '', description: '', command: '', schedule: '0 * * * *', is_active: true }

export default function Deployment() {
  const depsApi = useApi<Dep[]>(() => deployment.list() as Promise<Dep[]>, [], { pollInterval: 10000 })
  const tasksApi = useApi<ScheduledTask[]>(() => deployment.tasks.list() as Promise<ScheduledTask[]>, [], { pollInterval: 10000 })
  const [tab, setTab] = useState<'deployments' | 'tasks'>('deployments')

  // Deploy modal
  const [depModal, setDepModal] = useState(false)
  const [depForm, setDepForm] = useState<DeployForm>(emptyDeploy)
  const [depSaving, setDepSaving] = useState(false)

  // Task modal
  const [taskModal, setTaskModal] = useState(false)
  const [taskForm, setTaskForm] = useState<TaskForm>(emptyTask)
  const [taskSaving, setTaskSaving] = useState(false)

  const [runLoading, setRunLoading] = useState<number | null>(null)
  const [outputModal, setOutputModal] = useState<{ title: string; output: string } | null>(null)
  const [error, setError] = useState('')

  const saveDeploy = async () => {
    if (!depForm.name || !depForm.config) return
    setDepSaving(true)
    try {
      await deployment.create(depForm)
      setDepModal(false)
      setDepForm(emptyDeploy)
      depsApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setDepSaving(false) }
  }

  const runDeploy = async (id: number) => {
    setRunLoading(id)
    try {
      await deployment.run(id)
      await new Promise(r => setTimeout(r, 1000))
      depsApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setRunLoading(null) }
  }

  const deleteDeploy = async (id: number) => {
    if (!confirm('Delete this deployment?')) return
    await deployment.delete(id)
    depsApi.refetch()
  }

  const saveTask = async () => {
    if (!taskForm.name || !taskForm.command || !taskForm.schedule) return
    setTaskSaving(true)
    try {
      await deployment.tasks.create(taskForm)
      setTaskModal(false)
      setTaskForm(emptyTask)
      tasksApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setTaskSaving(false) }
  }

  const runTask = async (id: number) => {
    setRunLoading(id)
    try {
      await deployment.tasks.run(id)
      await new Promise(r => setTimeout(r, 1000))
      tasksApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setRunLoading(null) }
  }

  const deleteTask = async (id: number) => {
    if (!confirm('Delete this task?')) return
    await deployment.tasks.delete(id)
    tasksApi.refetch()
  }

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <Rocket className="w-6 h-6 text-amber-400" />
            Deployment & Automation
          </h1>
          <p className="text-sm text-slate-500 mt-0.5">Deploy applications, manage services, schedule tasks</p>
        </div>
        <button
          onClick={() => tab === 'deployments' ? setDepModal(true) : setTaskModal(true)}
          className="btn-primary"
        >
          <Plus className="w-4 h-4" />
          {tab === 'deployments' ? 'New Deployment' : 'New Task'}
        </button>
      </div>

      {error && (
        <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {error} <button onClick={() => setError('')} className="ml-2 underline">Dismiss</button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 bg-slate-900 border border-slate-800 rounded-xl p-1 w-fit">
        {(['deployments', 'tasks'] as const).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === t ? 'bg-fabrixe-600/20 text-fabrixe-300 border border-fabrixe-500/20'
                       : 'text-slate-500 hover:text-slate-300'
            }`}
          >
            {t === 'deployments' ? 'Deployments' : 'Scheduled Tasks'}
          </button>
        ))}
      </div>

      {/* ── Deployments ── */}
      {tab === 'deployments' && (
        <div className="space-y-3">
          {depsApi.loading ? <PageLoader /> :
            (depsApi.data ?? []).length === 0 ? (
              <EmptyState icon={Rocket} title="No deployments yet" description="Create a script, Docker, or systemd deployment."
                action={<button className="btn-primary" onClick={() => setDepModal(true)}><Plus className="w-4 h-4" />New Deployment</button>} />
            ) : (
              (depsApi.data ?? []).map(d => (
                <div key={d.id} className="card-sm">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-slate-200">{d.name}</span>
                        <StatusBadge status={d.status} />
                        <span className="badge-neutral text-xs">{d.deploy_type}</span>
                      </div>
                      {d.description && <p className="text-xs text-slate-500 mt-0.5">{d.description}</p>}
                      {d.last_run && <p className="text-xs text-slate-600 mt-1">Last run: {formatRelative(d.last_run)}</p>}
                      {d.last_output && (
                        <button
                          onClick={() => setOutputModal({ title: d.name, output: d.last_output })}
                          className="text-xs text-fabrixe-400 hover:text-fabrixe-300 mt-1"
                        >
                          View output →
                        </button>
                      )}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <button
                        onClick={() => runDeploy(d.id)}
                        disabled={d.status === 'running' || runLoading === d.id}
                        className="btn-success"
                      >
                        {runLoading === d.id ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                        Run
                      </button>
                      <button onClick={() => deleteDeploy(d.id)} className="btn-danger p-2">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )
          }
        </div>
      )}

      {/* ── Tasks ── */}
      {tab === 'tasks' && (
        <div className="space-y-3">
          {tasksApi.loading ? <PageLoader /> :
            (tasksApi.data ?? []).length === 0 ? (
              <EmptyState icon={CalendarClock} title="No scheduled tasks" description="Schedule recurring commands with cron expressions."
                action={<button className="btn-primary" onClick={() => setTaskModal(true)}><Plus className="w-4 h-4" />New Task</button>} />
            ) : (
              (tasksApi.data ?? []).map(t => (
                <div key={t.id} className="card-sm">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-slate-200">{t.name}</span>
                        <StatusBadge status={t.last_status} />
                        {!t.is_active && <span className="badge-neutral">Disabled</span>}
                      </div>
                      <p className="text-xs font-mono text-slate-500 mt-0.5">{t.schedule}</p>
                      <p className="text-xs font-mono text-slate-600 mt-0.5 truncate">{t.command}</p>
                      {t.last_run && <p className="text-xs text-slate-600 mt-1">Last: {formatRelative(t.last_run)}</p>}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <button onClick={() => runTask(t.id)} disabled={t.last_status === 'running' || runLoading === t.id} className="btn-success">
                        {runLoading === t.id ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                        Run now
                      </button>
                      <button onClick={() => deleteTask(t.id)} className="btn-danger p-2">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )
          }
        </div>
      )}

      {/* Create Deployment Modal */}
      <Modal open={depModal} onClose={() => setDepModal(false)} title="New Deployment">
        <div className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input className="input" value={depForm.name} onChange={e => setDepForm(f => ({ ...f, name: e.target.value }))} placeholder="My Deployment" />
          </div>
          <div>
            <label className="label">Description</label>
            <input className="input" value={depForm.description} onChange={e => setDepForm(f => ({ ...f, description: e.target.value }))} placeholder="Optional description" />
          </div>
          <div>
            <label className="label">Type *</label>
            <select className="input" value={depForm.deploy_type} onChange={e => setDepForm(f => ({ ...f, deploy_type: e.target.value as 'script' | 'docker' | 'systemd' }))}>
              <option value="script">Shell Script</option>
              <option value="docker">Docker</option>
              <option value="systemd">Systemd</option>
            </select>
          </div>
          <div>
            <label className="label">
              {depForm.deploy_type === 'script' ? 'Shell Script *' :
               depForm.deploy_type === 'docker' ? 'Docker Args (e.g. run myimage) *' :
               'Service Name *'}
            </label>
            <textarea
              className="input min-h-[120px] font-mono text-xs resize-y"
              value={depForm.config}
              onChange={e => setDepForm(f => ({ ...f, config: e.target.value }))}
              placeholder={depForm.deploy_type === 'script' ? '#!/bin/bash\necho "deploying..."' :
                          depForm.deploy_type === 'docker' ? 'run -d --name myapp myimage:latest' :
                          'nginx.service'}
            />
          </div>
          <div className="flex justify-end gap-2">
            <button className="btn-secondary" onClick={() => setDepModal(false)}>Cancel</button>
            <button className="btn-primary" onClick={saveDeploy} disabled={depSaving}>
              {depSaving ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              Create
            </button>
          </div>
        </div>
      </Modal>

      {/* Create Task Modal */}
      <Modal open={taskModal} onClose={() => setTaskModal(false)} title="New Scheduled Task">
        <div className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input className="input" value={taskForm.name} onChange={e => setTaskForm(f => ({ ...f, name: e.target.value }))} placeholder="Daily backup" />
          </div>
          <div>
            <label className="label">Description</label>
            <input className="input" value={taskForm.description} onChange={e => setTaskForm(f => ({ ...f, description: e.target.value }))} />
          </div>
          <div>
            <label className="label">Command *</label>
            <input className="input font-mono text-xs" value={taskForm.command} onChange={e => setTaskForm(f => ({ ...f, command: e.target.value }))} placeholder="/opt/scripts/backup.sh" />
          </div>
          <div>
            <label className="label">Schedule (cron) *</label>
            <input className="input font-mono text-xs" value={taskForm.schedule} onChange={e => setTaskForm(f => ({ ...f, schedule: e.target.value }))} placeholder="0 2 * * *" />
            <p className="text-xs text-slate-600 mt-1">Format: minute hour day month weekday</p>
          </div>
          <div className="flex items-center gap-2">
            <input type="checkbox" id="task-active" checked={taskForm.is_active} onChange={e => setTaskForm(f => ({ ...f, is_active: e.target.checked }))} className="accent-fabrixe-600" />
            <label htmlFor="task-active" className="text-sm text-slate-300">Enable task</label>
          </div>
          <div className="flex justify-end gap-2">
            <button className="btn-secondary" onClick={() => setTaskModal(false)}>Cancel</button>
            <button className="btn-primary" onClick={saveTask} disabled={taskSaving}>
              {taskSaving ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              Create
            </button>
          </div>
        </div>
      </Modal>

      {/* Output Modal */}
      {outputModal && (
        <Modal open={!!outputModal} onClose={() => setOutputModal(null)} title={`Output: ${outputModal.title}`} size="lg">
          <pre className="text-xs font-mono text-slate-300 bg-slate-950 rounded-xl p-4 overflow-x-auto max-h-96 whitespace-pre-wrap">
            {outputModal.output || '(no output)'}
          </pre>
        </Modal>
      )}
    </div>
  )
}
