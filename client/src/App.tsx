import { useCallback, useEffect, useMemo, useState } from '@lynx-js/react'
import type { ReactNode } from 'react'

import { ApiError, createApiClient, resolveAssetUrl } from './api.js'
import type {
  ApiFieldErrors,
  Article,
  ArticlePayload,
  Certificate,
  CertificatePayload,
  CommonImage,
  CurrentFilter,
  FeaturedFilter,
  Intro,
  IntroPayload,
  Link,
  LinkListFilter,
  LinkPayload,
  ListFilter,
  Project,
  ProjectPayload,
  ResourceTab,
  Skill,
  SkillCategory,
  SkillCategoryPayload,
  SkillListFilter,
  SkillPayload,
  StarFilter,
  Status,
  StatusListFilter,
  Tool,
  ToolListFilter,
  ToolPayload,
  WorkExperience,
  WorkExperienceListFilter,
  WorkExperiencePayload,
} from './types.js'

import './App.css'

const TOKEN_KEY = 'portfolio_admin_token'
const STATUS_OPTIONS: Array<'all' | Status> = ['all', 'draft', 'published', 'archived']
const EDIT_STATUS_OPTIONS: Status[] = ['draft', 'published', 'archived']
const FEATURED_OPTIONS: FeaturedFilter[] = ['all', 'featured', 'standard']
const STAR_OPTIONS: StarFilter[] = ['all', 'starred', 'standard']
const CURRENT_OPTIONS: CurrentFilter[] = ['all', 'current', 'past']

interface ProjectForm {
  slug: string
  title: string
  summary: string
  description: string
  body: string
  techStack: string
  tags: string
  githubUrl: string
  demoUrl: string
  featured: boolean
  sortOrder: string
  status: Status
  imageAltText: string
  imageCaption: string
}

interface CertificateForm {
  slug: string
  title: string
  issuer: string
  summary: string
  description: string
  credentialUrl: string
  featured: boolean
  sortOrder: string
  status: Status
  issuedAt: string
  expiresAt: string
  imageAltText: string
  imageCaption: string
}

interface ArticleForm {
  title: string
  url: string
  source: string
  summary: string
  coverImageUrl: string
  featured: boolean
  sortOrder: string
  status: Status
  publishedAt: string
}

interface WorkExperienceForm {
  slug: string
  title: string
  company: string
  companyUrl: string
  companyLogoUrl: string
  employmentType: string
  location: string
  locationType: string
  summary: string
  description: string
  highlights: string
  responsibilities: string
  techStack: string
  skills: string
  startedAt: string
  endedAt: string
  current: boolean
  featured: boolean
  sortOrder: string
  status: Status
  publishedAt: string
}

interface LinkForm {
  label: string
  url: string
  iconClass: string
  sortOrder: string
  star: boolean
  status: Status
}

interface SkillCategoryForm {
  slug: string
  name: string
  description: string
  iconClass: string
  sortOrder: string
  status: Status
}

interface SkillForm {
  categoryId: string
  name: string
  summary: string
  iconClass: string
  sortOrder: string
  featured: boolean
  status: Status
}

interface ToolForm {
  name: string
  category: string
  summary: string
  iconClass: string
  tags: string
  sortOrder: string
  featured: boolean
  status: Status
}

interface IntroForm {
  title: string
  description: string
  imageAltText: string
  imageCaption: string
}

type Operation = 'idle' | 'saving' | 'deleting' | 'uploading'
type Mode = 'create' | 'edit'

const emptyContentFilter: ListFilter = {
  q: '',
  status: 'all',
  featured: 'all',
}

const emptyLinkFilter: LinkListFilter = {
  q: '',
  status: 'all',
  star: 'all',
}

const emptyWorkFilter: WorkExperienceListFilter = {
  q: '',
  status: 'all',
  featured: 'all',
  current: 'all',
}

const emptyStatusFilter: StatusListFilter = {
  q: '',
  status: 'all',
}

const emptySkillFilter: SkillListFilter = {
  q: '',
  status: 'all',
  featured: 'all',
  category_id: '',
  category: '',
}

const emptyToolFilter: ToolListFilter = {
  q: '',
  status: 'all',
  featured: 'all',
  category: '',
  tag: '',
}

const emptyProjectForm: ProjectForm = {
  slug: '',
  title: '',
  summary: '',
  description: '',
  body: '',
  techStack: '',
  tags: '',
  githubUrl: '',
  demoUrl: '',
  featured: false,
  sortOrder: '0',
  status: 'draft',
  imageAltText: '',
  imageCaption: '',
}

const emptyCertificateForm: CertificateForm = {
  slug: '',
  title: '',
  issuer: '',
  summary: '',
  description: '',
  credentialUrl: '',
  featured: false,
  sortOrder: '0',
  status: 'draft',
  issuedAt: '',
  expiresAt: '',
  imageAltText: '',
  imageCaption: '',
}

const emptyArticleForm: ArticleForm = {
  title: '',
  url: '',
  source: '',
  summary: '',
  coverImageUrl: '',
  featured: false,
  sortOrder: '0',
  status: 'draft',
  publishedAt: '',
}

const emptyWorkExperienceForm: WorkExperienceForm = {
  slug: '',
  title: '',
  company: '',
  companyUrl: '',
  companyLogoUrl: '',
  employmentType: '',
  location: '',
  locationType: '',
  summary: '',
  description: '',
  highlights: '',
  responsibilities: '',
  techStack: '',
  skills: '',
  startedAt: '',
  endedAt: '',
  current: false,
  featured: false,
  sortOrder: '0',
  status: 'draft',
  publishedAt: '',
}

const emptyLinkForm: LinkForm = {
  label: '',
  url: '',
  iconClass: '',
  sortOrder: '0',
  star: false,
  status: 'draft',
}

const emptySkillCategoryForm: SkillCategoryForm = {
  slug: '',
  name: '',
  description: '',
  iconClass: '',
  sortOrder: '0',
  status: 'draft',
}

const emptySkillForm: SkillForm = {
  categoryId: '',
  name: '',
  summary: '',
  iconClass: '',
  sortOrder: '0',
  featured: false,
  status: 'draft',
}

const emptyToolForm: ToolForm = {
  name: '',
  category: '',
  summary: '',
  iconClass: '',
  tags: '',
  sortOrder: '0',
  featured: false,
  status: 'draft',
}

const emptyIntroForm: IntroForm = {
  title: '',
  description: '',
  imageAltText: '',
  imageCaption: '',
}

type AppRoute = 'public' | 'admin'
type PublicTheme = 'light' | 'dark'

interface PublicData {
  intro: Intro | null
  projects: Project[]
  workExperiences: WorkExperience[]
  links: Link[]
}

const emptyPublicData: PublicData = {
  intro: null,
  projects: [],
  workExperiences: [],
  links: [],
}

export function App() {
  const [route, setRoute] = useState<AppRoute>(readInitialRoute)

  useEffect(() => {
    const windowRef = browserWindow()
    if (!windowRef) return

    const syncRoute = () => setRoute(readInitialRoute())
    windowRef.addEventListener?.('popstate', syncRoute)
    return () => windowRef.removeEventListener?.('popstate', syncRoute)
  }, [])

  const navigate = useCallback((nextRoute: AppRoute) => {
    const path = nextRoute === 'admin' ? '/login' : '/'
    pushPath(path)
    setRoute(nextRoute)
  }, [])

  if (route === 'admin') {
    return <AdminApp onViewPublic={() => navigate('public')} />
  }

  return <PublicHome />
}

function PublicHome() {
  const [data, setData] = useState<PublicData>(emptyPublicData)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [theme, setTheme] = useState<PublicTheme>('light')
  const api = useMemo(() => createApiClient(() => null), [])

  useEffect(() => {
    injectOnestFontLinks()
  }, [])

  useEffect(() => {
    let cancelled = false

    async function loadPublicData() {
      setLoading(true)
      setError('')
      const [
        intro,
        projects,
        workExperiences,
        links,
      ] = await Promise.all([
        optionalResult(api.getIntro(), null),
        optionalResult(api.listProjects({ q: '', status: 'published', featured: 'all' }), []),
        optionalResult(api.listWorkExperiences({ q: '', status: 'published', featured: 'all', current: 'all' }), []),
        optionalResult(api.listLinks({ q: '', status: 'published', star: 'all' }), []),
      ])

      if (cancelled) return

      setData({
        intro,
        projects,
        workExperiences,
        links,
      })
      if (!intro && projects.length === 0 && workExperiences.length === 0) {
        setError('Published portfolio content is not available yet.')
      }
      setLoading(false)
    }

    void loadPublicData().catch(nextError => {
      if (cancelled) return
      setError(nextError instanceof Error ? nextError.message : 'Unable to load portfolio content.')
      setLoading(false)
    })

    return () => {
      cancelled = true
    }
  }, [api])

  const visibleProjects = [...data.projects].sort(byFeaturedThenProjectOrder).slice(0, 3)
  const starredLinks = data.links.filter(link => link.star).slice(0, 4)
  const nextTheme = theme === 'dark' ? 'light' : 'dark'

  return (
    <scroll-view className={theme === 'dark' ? 'PublicPage PublicPage--dark' : 'PublicPage PublicPage--light'} scroll-y>
      <view className='PublicThemeToggle' bindtap={() => setTheme(nextTheme)}>
        <text className='PublicThemeToggleText'>{nextTheme === 'dark' ? 'Dark mode' : 'Light mode'}</text>
      </view>
      <view className='PublicHero'>
        <view className='PublicHeroCopy'>
          <text className='PublicEyebrow'>AI ENGINEER</text>
          <text className='PublicTitle'>{data.intro?.title || 'AI Engineer'}</text>
          <text className='PublicLead'>
            {data.intro?.description || 'I build AI-enabled products, backend systems, and polished interfaces that turn model capabilities into reliable user workflows.'}
          </text>
          <view className='PublicHeroActions'>
            {starredLinks.map(link => (
              <TapButton key={link.id} label={link.label} onTap={() => openExternal(link.url)} variant='primary' />
            ))}
          </view>
        </view>
        <view className='PublicPortraitWrap'>
          {canRenderPublicImage(data.intro?.profile_picture?.url) ? (
            <image src={resolveAssetUrl(data.intro!.profile_picture!.url)} className='PublicPortrait' />
          ) : (
            <view className='PublicPortraitFallback'>
              <text className='PublicPortraitInitials'>EA</text>
            </view>
          )}
        </view>
      </view>

      {loading ? <StateBlock title='Loading portfolio...' /> : null}
      {error ? <view className='PublicNotice'><text className='MutedText'>{error}</text></view> : null}

      <PublicSection title='Projects' subtitle='Featured shipped systems and product-facing engineering.'>
        <view className='PublicCardGrid'>
          {visibleProjects.map(project => <ProjectCard key={project.id} project={project} />)}
          {!loading && visibleProjects.length === 0 ? <StateBlock title='No published projects yet.' compact /> : null}
        </view>
      </PublicSection>

      <PublicSection title='Experience' subtitle='Recent work focused on backend APIs, AI workflows, and production delivery.'>
        <view className='PublicTimeline'>
          {data.workExperiences.slice(0, 4).map(experience => (
            <view key={experience.id} className='PublicTimelineItem'>
              <text className='PublicPanelTitle'>{experience.title}</text>
              <text className='PublicPanelMeta'>{experience.company} - {dateRange(experience.started_at, experience.ended_at, experience.current)}</text>
              <text className='PublicPanelText'>{experience.summary || experience.description || 'Engineering delivery role.'}</text>
              <view className='TagRow'>
                {(experience.skills ?? experience.tech_stack ?? []).slice(0, 5).map(item => <text key={item} className='PublicPill'>{item}</text>)}
              </view>
            </view>
          ))}
          {!loading && data.workExperiences.length === 0 ? <StateBlock title='No published experience yet.' compact /> : null}
        </view>
      </PublicSection>
    </scroll-view>
  )
}

function AdminApp({ onViewPublic }: { onViewPublic: () => void }) {
  const [token, setTokenState] = useState(readToken)
  const [loginUsername, setLoginUsername] = useState('admin')
  const [loginPassword, setLoginPassword] = useState('')
  const [activeTab, setActiveTab] = useState<ResourceTab>('projects')
  const [projectFilter, setProjectFilter] = useState<ListFilter>(emptyContentFilter)
  const [certificateFilter, setCertificateFilter] = useState<ListFilter>(emptyContentFilter)
  const [articleFilter, setArticleFilter] = useState<ListFilter>(emptyContentFilter)
  const [workFilter, setWorkFilter] = useState<WorkExperienceListFilter>(emptyWorkFilter)
  const [linkFilter, setLinkFilter] = useState<LinkListFilter>(emptyLinkFilter)
  const [skillCategoryFilter, setSkillCategoryFilter] = useState<StatusListFilter>(emptyStatusFilter)
  const [skillFilter, setSkillFilter] = useState<SkillListFilter>(emptySkillFilter)
  const [toolFilter, setToolFilter] = useState<ToolListFilter>(emptyToolFilter)
  const [projects, setProjects] = useState<Project[]>([])
  const [certificates, setCertificates] = useState<Certificate[]>([])
  const [articles, setArticles] = useState<Article[]>([])
  const [workExperiences, setWorkExperiences] = useState<WorkExperience[]>([])
  const [links, setLinks] = useState<Link[]>([])
  const [skillCategories, setSkillCategories] = useState<SkillCategory[]>([])
  const [skills, setSkills] = useState<Skill[]>([])
  const [tools, setTools] = useState<Tool[]>([])
  const [intro, setIntro] = useState<Intro | null>(null)
  const [selectedProjectID, setSelectedProjectID] = useState<string | null>(null)
  const [selectedCertificateID, setSelectedCertificateID] = useState<string | null>(null)
  const [selectedArticleID, setSelectedArticleID] = useState<string | null>(null)
  const [selectedWorkExperienceID, setSelectedWorkExperienceID] = useState<string | null>(null)
  const [selectedLinkID, setSelectedLinkID] = useState<string | null>(null)
  const [selectedSkillCategoryID, setSelectedSkillCategoryID] = useState<string | null>(null)
  const [selectedSkillID, setSelectedSkillID] = useState<string | null>(null)
  const [selectedToolID, setSelectedToolID] = useState<string | null>(null)
  const [projectForm, setProjectForm] = useState<ProjectForm>(emptyProjectForm)
  const [certificateForm, setCertificateForm] = useState<CertificateForm>(emptyCertificateForm)
  const [articleForm, setArticleForm] = useState<ArticleForm>(emptyArticleForm)
  const [workExperienceForm, setWorkExperienceForm] = useState<WorkExperienceForm>(emptyWorkExperienceForm)
  const [linkForm, setLinkForm] = useState<LinkForm>(emptyLinkForm)
  const [skillCategoryForm, setSkillCategoryForm] = useState<SkillCategoryForm>(emptySkillCategoryForm)
  const [skillForm, setSkillForm] = useState<SkillForm>(emptySkillForm)
  const [toolForm, setToolForm] = useState<ToolForm>(emptyToolForm)
  const [introForm, setIntroForm] = useState<IntroForm>(emptyIntroForm)
  const [projectMode, setProjectMode] = useState<Mode>('create')
  const [certificateMode, setCertificateMode] = useState<Mode>('create')
  const [articleMode, setArticleMode] = useState<Mode>('create')
  const [workExperienceMode, setWorkExperienceMode] = useState<Mode>('create')
  const [linkMode, setLinkMode] = useState<Mode>('create')
  const [skillCategoryMode, setSkillCategoryMode] = useState<Mode>('create')
  const [skillMode, setSkillMode] = useState<Mode>('create')
  const [toolMode, setToolMode] = useState<Mode>('create')
  const [loading, setLoading] = useState(false)
  const [operation, setOperation] = useState<Operation>('idle')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<ApiFieldErrors>({})

  const api = useMemo(() => createApiClient(() => token), [token])
  const selectedProject = projects.find(project => project.id === selectedProjectID) ?? null
  const selectedCertificate = certificates.find(certificate => certificate.id === selectedCertificateID) ?? null
  const selectedArticle = articles.find(article => article.id === selectedArticleID) ?? null
  const selectedWorkExperience = workExperiences.find(experience => experience.id === selectedWorkExperienceID) ?? null
  const selectedLink = links.find(link => link.id === selectedLinkID) ?? null
  const selectedSkillCategory = skillCategories.find(category => category.id === selectedSkillCategoryID) ?? null
  const selectedSkill = skills.find(skill => skill.id === selectedSkillID) ?? null
  const selectedTool = tools.find(tool => tool.id === selectedToolID) ?? null

  const setToken = useCallback((nextToken: string) => {
    writeToken(nextToken)
    setTokenState(nextToken)
  }, [])

  const clearError = useCallback(() => {
    setError('')
    setFieldErrors({})
  }, [])

  const loadProjects = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const next = await api.listProjects(projectFilter)
      setProjects(next)
      if (next.length === 0) {
        newProjectState(setSelectedProjectID, setProjectForm, setProjectMode)
      } else if (!next.some(item => item.id === selectedProjectID)) {
        selectProjectState(next[0], setSelectedProjectID, setProjectForm, setProjectMode)
      }
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setLoading(false)
    }
  }, [api, clearError, projectFilter, selectedProjectID])

  const loadCertificates = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const next = await api.listCertificates(certificateFilter)
      setCertificates(next)
      if (next.length === 0) {
        newCertificateState(setSelectedCertificateID, setCertificateForm, setCertificateMode)
      } else if (!next.some(item => item.id === selectedCertificateID)) {
        selectCertificateState(next[0], setSelectedCertificateID, setCertificateForm, setCertificateMode)
      }
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setLoading(false)
    }
  }, [api, certificateFilter, clearError, selectedCertificateID])

  const loadArticles = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const next = await api.listArticles(articleFilter)
      setArticles(next)
      if (next.length === 0) {
        newArticleState(setSelectedArticleID, setArticleForm, setArticleMode)
      } else if (!next.some(item => item.id === selectedArticleID)) {
        selectArticleState(next[0], setSelectedArticleID, setArticleForm, setArticleMode)
      }
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setLoading(false)
    }
  }, [api, articleFilter, clearError, selectedArticleID])

  const loadLinks = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const next = await api.listLinks(linkFilter)
      setLinks(next)
      if (next.length === 0) {
        newLinkState(setSelectedLinkID, setLinkForm, setLinkMode)
      } else if (!next.some(item => item.id === selectedLinkID)) {
        selectLinkState(next[0], setSelectedLinkID, setLinkForm, setLinkMode)
      }
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setLoading(false)
    }
  }, [api, clearError, linkFilter, selectedLinkID])

  const loadIntro = useCallback(async () => {
    setLoading(true)
    clearError()
    try {
      const next = await api.getIntro()
      setIntro(next)
      setIntroForm(introToForm(next))
    } catch (nextError) {
      setIntro(null)
      setIntroForm(emptyIntroForm)
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setLoading(false)
    }
  }, [api, clearError])

  const loadActive = useCallback(() => {
    switch (activeTab) {
      case 'projects':
        return loadProjects()
      case 'certificates':
        return loadCertificates()
      case 'articles':
        return loadArticles()
      case 'links':
        return loadLinks()
      case 'intro':
        return loadIntro()
    }
  }, [activeTab, loadArticles, loadCertificates, loadIntro, loadLinks, loadProjects])

  useEffect(() => {
    if (token) {
      void loadActive()
    }
  }, [loadActive, token])

  const onLogin = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const response = await api.login(loginUsername.trim(), loginPassword)
      setToken(response.token)
      setLoginPassword('')
      setActiveTab('projects')
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loginPassword, loginUsername, setToken])

  const onLogout = useCallback(() => {
    writeToken('')
    setTokenState('')
    setProjects([])
    setCertificates([])
    setArticles([])
    setLinks([])
    setIntro(null)
    newProjectState(setSelectedProjectID, setProjectForm, setProjectMode)
    newCertificateState(setSelectedCertificateID, setCertificateForm, setCertificateMode)
    newArticleState(setSelectedArticleID, setArticleForm, setArticleMode)
    newLinkState(setSelectedLinkID, setLinkForm, setLinkMode)
    setIntroForm(emptyIntroForm)
    clearError()
  }, [clearError])

  const saveProject = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const payload = projectPayload(projectForm)
      const saved = projectMode === 'create' || !selectedProject
        ? await api.createProject(payload)
        : await api.updateProject(selectedProject.id, payload)
      selectProjectState(saved, setSelectedProjectID, setProjectForm, setProjectMode)
      await loadProjects()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadProjects, projectForm, projectMode, selectedProject])

  const deleteProject = useCallback(async () => {
    if (!selectedProject || !confirmAction(`Delete project "${selectedProject.title}"?`)) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      await api.deleteProject(selectedProject.id)
      newProjectState(setSelectedProjectID, setProjectForm, setProjectMode)
      await loadProjects()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadProjects, selectedProject])

  const saveCertificate = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const payload = certificatePayload(certificateForm)
      const saved = certificateMode === 'create' || !selectedCertificate
        ? await api.createCertificate(payload)
        : await api.updateCertificate(selectedCertificate.id, payload)
      selectCertificateState(saved, setSelectedCertificateID, setCertificateForm, setCertificateMode)
      await loadCertificates()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, certificateForm, certificateMode, clearError, loadCertificates, selectedCertificate])

  const deleteCertificate = useCallback(async () => {
    if (!selectedCertificate || !confirmAction(`Delete certificate "${selectedCertificate.title}"?`)) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      await api.deleteCertificate(selectedCertificate.id)
      newCertificateState(setSelectedCertificateID, setCertificateForm, setCertificateMode)
      await loadCertificates()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadCertificates, selectedCertificate])

  const saveArticle = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const payload = articlePayload(articleForm)
      const saved = articleMode === 'create' || !selectedArticle
        ? await api.createArticle(payload)
        : await api.updateArticle(selectedArticle.id, payload)
      selectArticleState(saved, setSelectedArticleID, setArticleForm, setArticleMode)
      await loadArticles()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, articleForm, articleMode, clearError, loadArticles, selectedArticle])

  const deleteArticle = useCallback(async () => {
    if (!selectedArticle || !confirmAction(`Delete article "${selectedArticle.title}"?`)) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      await api.deleteArticle(selectedArticle.id)
      newArticleState(setSelectedArticleID, setArticleForm, setArticleMode)
      await loadArticles()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadArticles, selectedArticle])

  const saveLink = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const payload = linkPayload(linkForm)
      const saved = linkMode === 'create' || !selectedLink
        ? await api.createLink(payload)
        : await api.updateLink(selectedLink.id, payload)
      selectLinkState(saved, setSelectedLinkID, setLinkForm, setLinkMode)
      await loadLinks()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, linkForm, linkMode, loadLinks, selectedLink])

  const deleteLink = useCallback(async () => {
    if (!selectedLink || !confirmAction(`Delete link "${selectedLink.label}"?`)) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      await api.deleteLink(selectedLink.id)
      newLinkState(setSelectedLinkID, setLinkForm, setLinkMode)
      await loadLinks()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadLinks, selectedLink])

  const saveIntro = useCallback(async () => {
    clearError()
    setOperation('saving')
    try {
      const saved = await api.updateIntro(introPayload(introForm))
      setIntro(saved)
      setIntroForm(introToForm(saved))
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, introForm])

  const deleteIntro = useCallback(async () => {
    if (!confirmAction('Delete intro content?')) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      await api.deleteIntro()
      setIntro(null)
      setIntroForm(emptyIntroForm)
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError])

  const uploadProjectImage = useCallback(async (variant: 'main' | 'gallery') => {
    if (!selectedProject) {
      setError('Save the project before uploading an image.')
      return
    }
    clearError()
    setOperation('uploading')
    try {
      const file = await pickImageFile()
      const updated = variant === 'main'
        ? await api.uploadProjectMainImage(selectedProject.id, file, projectForm.imageAltText, projectForm.imageCaption)
        : await api.uploadProjectGalleryImage(selectedProject.id, file, projectForm.imageAltText, projectForm.imageCaption)
      selectProjectState(updated, setSelectedProjectID, setProjectForm, setProjectMode)
      await loadProjects()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadProjects, projectForm.imageAltText, projectForm.imageCaption, selectedProject])

  const deleteProjectImage = useCallback(async (imageID: string) => {
    if (!selectedProject || !confirmAction('Delete this project image?')) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      const updated = await api.deleteProjectImage(selectedProject.id, imageID)
      selectProjectState(updated, setSelectedProjectID, setProjectForm, setProjectMode)
      await loadProjects()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadProjects, selectedProject])

  const uploadCertificateImage = useCallback(async () => {
    if (!selectedCertificate) {
      setError('Save the certificate before uploading an image.')
      return
    }
    clearError()
    setOperation('uploading')
    try {
      const file = await pickImageFile()
      const updated = await api.uploadCertificateImage(
        selectedCertificate.id,
        file,
        certificateForm.imageAltText,
        certificateForm.imageCaption,
      )
      selectCertificateState(updated, setSelectedCertificateID, setCertificateForm, setCertificateMode)
      await loadCertificates()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, certificateForm.imageAltText, certificateForm.imageCaption, clearError, loadCertificates, selectedCertificate])

  const deleteCertificateImage = useCallback(async () => {
    if (!selectedCertificate?.image || !confirmAction('Delete this certificate image?')) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      const updated = await api.deleteCertificateImage(selectedCertificate.id)
      selectCertificateState(updated, setSelectedCertificateID, setCertificateForm, setCertificateMode)
      await loadCertificates()
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, loadCertificates, selectedCertificate])

  const uploadIntroPicture = useCallback(async () => {
    clearError()
    setOperation('uploading')
    try {
      const file = await pickImageFile()
      const updated = await api.uploadIntroProfilePicture(file, introForm.imageAltText, introForm.imageCaption)
      setIntro(updated)
      setIntroForm(introToForm(updated))
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, introForm.imageAltText, introForm.imageCaption])

  const deleteIntroPicture = useCallback(async () => {
    if (!intro?.profile_picture || !confirmAction('Delete this profile picture?')) {
      return
    }
    clearError()
    setOperation('deleting')
    try {
      const updated = await api.deleteIntroProfilePicture()
      setIntro(updated)
      setIntroForm(introToForm(updated))
    } catch (nextError) {
      handleError(nextError, setError, setFieldErrors)
    } finally {
      setOperation('idle')
    }
  }, [api, clearError, intro])

  if (!token) {
    return (
      <view className='LoginScreen'>
        <view className='LoginPanel'>
          <text className='Eyebrow'>PORTFOLIO ADMIN</text>
          <text className='LoginTitle'>Sign in</text>
          <view className='FieldStack'>
            <TextField label='Username' value={loginUsername} placeholder='admin' onChange={setLoginUsername} />
            <TextField label='Password' value={loginPassword} placeholder='change-me' secure onChange={setLoginPassword} />
          </view>
          {error ? <ErrorPanel error={error} fields={fieldErrors} /> : null}
          <TapButton label={operation === 'saving' ? 'Signing in...' : 'Sign in'} onTap={onLogin} variant='primary' disabled={operation !== 'idle'} />
          <TapButton label='View public profile' onTap={onViewPublic} variant='ghost' />
        </view>
      </view>
    )
  }

  return (
    <view className='AppRoot'>
      <view className='TopBar'>
        <view>
          <text className='Eyebrow'>PORTFOLIO ADMIN</text>
          <text className='AppTitle'>Content workspace</text>
        </view>
        <view className='TopActions'>
          <text className='MutedText'>{operation === 'idle' ? 'Ready' : operation}</text>
          <TapButton label='View public' onTap={onViewPublic} variant='secondary' />
          <TapButton label='Log out' onTap={onLogout} variant='ghost' />
        </view>
      </view>

      <view className='Toolbar'>
        <Segmented
          options={[
            { value: 'projects', label: `Projects ${projects.length}` },
            { value: 'certificates', label: `Certificates ${certificates.length}` },
            { value: 'articles', label: `Articles ${articles.length}` },
            { value: 'links', label: `Links ${links.length}` },
            { value: 'intro', label: 'Intro' },
          ]}
          value={activeTab}
          onChange={value => {
            setActiveTab(value)
            clearError()
          }}
        />
        <view className='ToolbarSpacer' />
        {activeTab !== 'intro' ? (
          <TapButton label={newButtonLabel(activeTab)} onTap={() => newActiveRecord(activeTab, {
            setSelectedProjectID,
            setProjectForm,
            setProjectMode,
            setSelectedCertificateID,
            setCertificateForm,
            setCertificateMode,
            setSelectedArticleID,
            setArticleForm,
            setArticleMode,
            setSelectedLinkID,
            setLinkForm,
            setLinkMode,
            clearError,
          })} variant='primary' />
        ) : null}
        <TapButton label='Refresh' onTap={() => void loadActive()} variant='secondary' />
      </view>

      <view className='Workspace'>
        <view className='ListPane'>
          {activeTab === 'links' ? (
            <LinkFilterBar filter={linkFilter} onFilterChange={setLinkFilter} />
          ) : activeTab === 'intro' ? (
            <IntroSummary intro={intro} loading={loading} />
          ) : (
            <ContentFilterBar
              filter={contentFilterFor(activeTab, projectFilter, certificateFilter, articleFilter)}
              onFilterChange={filter => {
                if (activeTab === 'projects') setProjectFilter(filter)
                if (activeTab === 'certificates') setCertificateFilter(filter)
                if (activeTab === 'articles') setArticleFilter(filter)
              }}
            />
          )}
          {loading ? <StateBlock title='Loading records...' /> : null}
          {!loading && activeTab === 'projects' ? <ProjectList projects={projects} selectedID={selectedProjectID} onSelect={project => {
            selectProjectState(project, setSelectedProjectID, setProjectForm, setProjectMode)
            clearError()
          }} /> : null}
          {!loading && activeTab === 'certificates' ? <CertificateList certificates={certificates} selectedID={selectedCertificateID} onSelect={certificate => {
            selectCertificateState(certificate, setSelectedCertificateID, setCertificateForm, setCertificateMode)
            clearError()
          }} /> : null}
          {!loading && activeTab === 'articles' ? <ArticleList articles={articles} selectedID={selectedArticleID} onSelect={article => {
            selectArticleState(article, setSelectedArticleID, setArticleForm, setArticleMode)
            clearError()
          }} /> : null}
          {!loading && activeTab === 'links' ? <LinkList links={links} selectedID={selectedLinkID} onSelect={link => {
            selectLinkState(link, setSelectedLinkID, setLinkForm, setLinkMode)
            clearError()
          }} /> : null}
        </view>

        <scroll-view className='EditorPane' scroll-y>
          {error ? <ErrorPanel error={error} fields={fieldErrors} /> : null}
          {activeTab === 'projects' ? (
            <ProjectEditor
              form={projectForm}
              mode={projectMode}
              selected={selectedProject}
              operation={operation}
              onFormChange={setProjectForm}
              onSave={saveProject}
              onDelete={deleteProject}
              onUploadMain={() => void uploadProjectImage('main')}
              onUploadGallery={() => void uploadProjectImage('gallery')}
              onDeleteImage={deleteProjectImage}
            />
          ) : null}
          {activeTab === 'certificates' ? (
            <CertificateEditor
              form={certificateForm}
              mode={certificateMode}
              selected={selectedCertificate}
              operation={operation}
              onFormChange={setCertificateForm}
              onSave={saveCertificate}
              onDelete={deleteCertificate}
              onUploadImage={uploadCertificateImage}
              onDeleteImage={deleteCertificateImage}
            />
          ) : null}
          {activeTab === 'articles' ? (
            <ArticleEditor
              form={articleForm}
              mode={articleMode}
              selected={selectedArticle}
              operation={operation}
              onFormChange={setArticleForm}
              onSave={saveArticle}
              onDelete={deleteArticle}
            />
          ) : null}
          {activeTab === 'links' ? (
            <LinkEditor
              form={linkForm}
              mode={linkMode}
              selected={selectedLink}
              operation={operation}
              onFormChange={setLinkForm}
              onSave={saveLink}
              onDelete={deleteLink}
            />
          ) : null}
          {activeTab === 'intro' ? (
            <IntroEditor
              form={introForm}
              intro={intro}
              operation={operation}
              onFormChange={setIntroForm}
              onSave={saveIntro}
              onDelete={deleteIntro}
              onUploadPicture={uploadIntroPicture}
              onDeletePicture={deleteIntroPicture}
            />
          ) : null}
        </scroll-view>
      </view>
    </view>
  )
}

function PublicSection({ title, subtitle, children }: { title: string, subtitle: string, children: ReactNode }) {
  return (
    <view className='PublicSection'>
      <view className='PublicSectionHeader'>
        <text className='PublicSectionTitle'>{title}</text>
        <text className='PublicSectionSubtitle'>{subtitle}</text>
      </view>
      {children}
    </view>
  )
}

function ProjectCard({ project }: { project: Project }) {
  return (
    <view className='PublicProjectCard' bindtap={() => openExternal(project.demo_url || project.github_url || '')}>
      {canRenderPublicImage(project.main_image?.url) ? (
        <image src={resolveAssetUrl(project.main_image!.url)} className='PublicProjectImage' />
      ) : (
        <view className='PublicProjectImage PublicProjectImage--empty'>
          <text className='PublicProjectInitial'>{project.title.slice(0, 1).toUpperCase()}</text>
        </view>
      )}
      <view className='PublicProjectBody'>
        <view className='RecordHeader'>
          <text className='PublicPanelTitle'>{project.title}</text>
          {project.featured ? <text className='PublicPill PublicPill--featured'>Featured</text> : null}
        </view>
        <text className='PublicPanelText'>{project.summary || project.description || 'Published project.'}</text>
        <view className='TagRow'>
          {(project.tech_stack ?? project.tags ?? []).slice(0, 5).map(item => <text key={item} className='PublicPill'>{item}</text>)}
        </view>
      </view>
    </view>
  )
}

interface ButtonProps {
  label: string
  onTap: () => void
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  disabled?: boolean
}

function TapButton({ label, onTap, variant = 'secondary', disabled = false }: ButtonProps) {
  return (
    <text
      className={`Button Button--${variant}${disabled ? ' Button--disabled' : ''}`}
      bindtap={() => {
        if (!disabled) {
          onTap()
        }
      }}
    >
      {label}
    </text>
  )
}

interface TextFieldProps {
  label: string
  value: string
  placeholder?: string
  multiline?: boolean
  secure?: boolean
  numeric?: boolean
  onChange: (value: string) => void
}

function TextField({ label, value, placeholder, multiline = false, secure = false, numeric = false, onChange }: TextFieldProps) {
  const fieldID = `field-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${multiline ? 'area' : 'input'}`

  useEffect(() => {
    setElementValue(fieldID, value)
  }, [fieldID, value])

  return (
    <view className={multiline ? 'Field Field--wide' : 'Field'}>
      <text className='FieldLabel'>{label}</text>
      {multiline ? (
        <textarea id={fieldID} className='TextArea' placeholder={placeholder} bindinput={event => onChange(eventValue(event))} />
      ) : (
        <input
          id={fieldID}
          className='Input'
          type={secure ? 'password' : numeric ? 'number' : 'text'}
          placeholder={placeholder}
          bindinput={event => onChange(eventValue(event))}
        />
      )}
    </view>
  )
}

interface ToggleProps {
  label: string
  value: boolean
  onLabel?: string
  offLabel?: string
  onChange: (value: boolean) => void
}

function ToggleField({ label, value, onLabel = 'Enabled', offLabel = 'Disabled', onChange }: ToggleProps) {
  return (
    <view className='ToggleField'>
      <text className='FieldLabel'>{label}</text>
      <view className='ToggleTrack' bindtap={() => onChange(!value)}>
        <view className={value ? 'ToggleKnob ToggleKnob--on' : 'ToggleKnob'} />
        <text className='ToggleText'>{value ? onLabel : offLabel}</text>
      </view>
    </view>
  )
}

interface SegmentedProps<T extends string> {
  options: Array<{ value: T, label: string }>
  value: T
  onChange: (value: T) => void
}

function Segmented<T extends string>({ options, value, onChange }: SegmentedProps<T>) {
  return (
    <view className='Segmented'>
      {options.map(option => (
        <text
          key={option.value}
          className={option.value === value ? 'SegmentedItem SegmentedItem--active' : 'SegmentedItem'}
          bindtap={() => onChange(option.value)}
        >
          {option.label}
        </text>
      ))}
    </view>
  )
}

function ContentFilterBar({ filter, onFilterChange }: { filter: ListFilter, onFilterChange: (filter: ListFilter) => void }) {
  return (
    <view className='Filters'>
      <TextField label='Search' value={filter.q} placeholder='Title, slug, tag, issuer' onChange={q => onFilterChange({ ...filter, q })} />
      <view className='FilterRow'>
        <SelectGroup label='Status' value={filter.status} values={STATUS_OPTIONS} onChange={status => onFilterChange({ ...filter, status })} />
        <SelectGroup label='Featured' value={filter.featured} values={FEATURED_OPTIONS} onChange={featured => onFilterChange({ ...filter, featured })} />
      </view>
    </view>
  )
}

function LinkFilterBar({ filter, onFilterChange }: { filter: LinkListFilter, onFilterChange: (filter: LinkListFilter) => void }) {
  return (
    <view className='Filters'>
      <TextField label='Search' value={filter.q} placeholder='Label, URL, icon' onChange={q => onFilterChange({ ...filter, q })} />
      <view className='FilterRow'>
        <SelectGroup label='Status' value={filter.status} values={STATUS_OPTIONS} onChange={status => onFilterChange({ ...filter, status })} />
        <SelectGroup label='Star' value={filter.star} values={STAR_OPTIONS} onChange={star => onFilterChange({ ...filter, star })} />
      </view>
    </view>
  )
}

function SelectGroup<T extends string>({ label, value, values, onChange }: { label: string, value: T, values: T[], onChange: (value: T) => void }) {
  return (
    <view className='FilterGroup'>
      <text className='FieldLabel'>{label}</text>
      <Segmented options={values.map(item => ({ value: item, label: labelize(item) }))} value={value} onChange={onChange} />
    </view>
  )
}

function ProjectList({ projects, selectedID, onSelect }: { projects: Project[], selectedID: string | null, onSelect: (project: Project) => void }) {
  if (projects.length === 0) {
    return <StateBlock title='No projects match these filters.' />
  }
  return (
    <scroll-view className='RecordList' scroll-y>
      {projects.map(project => (
        <RecordItem
          key={project.id}
          active={project.id === selectedID}
          title={project.title}
          status={project.status}
          meta={`${project.slug} - order ${project.sort_order}`}
          summary={project.summary || project.description || 'No summary set'}
          pills={[project.featured ? 'featured' : '', ...(project.tech_stack ?? []).slice(0, 3)]}
          onTap={() => onSelect(project)}
        />
      ))}
    </scroll-view>
  )
}

function CertificateList({ certificates, selectedID, onSelect }: { certificates: Certificate[], selectedID: string | null, onSelect: (certificate: Certificate) => void }) {
  if (certificates.length === 0) {
    return <StateBlock title='No certificates match these filters.' />
  }
  return (
    <scroll-view className='RecordList' scroll-y>
      {certificates.map(certificate => (
        <RecordItem
          key={certificate.id}
          active={certificate.id === selectedID}
          title={certificate.title}
          status={certificate.status}
          meta={`${certificate.issuer} - order ${certificate.sort_order}`}
          summary={certificate.summary || certificate.description || 'No summary set'}
          pills={[certificate.featured ? 'featured' : '', certificate.issued_at ? `issued ${dateOnly(certificate.issued_at)}` : '']}
          onTap={() => onSelect(certificate)}
        />
      ))}
    </scroll-view>
  )
}

function ArticleList({ articles, selectedID, onSelect }: { articles: Article[], selectedID: string | null, onSelect: (article: Article) => void }) {
  if (articles.length === 0) {
    return <StateBlock title='No articles match these filters.' />
  }
  return (
    <scroll-view className='RecordList' scroll-y>
      {articles.map(article => (
        <RecordItem
          key={article.id}
          active={article.id === selectedID}
          title={article.title}
          status={article.status}
          meta={`${article.source} - order ${article.sort_order}`}
          summary={article.summary || article.url}
          pills={[article.featured ? 'featured' : '', article.published_at ? `published ${dateOnly(article.published_at)}` : '']}
          onTap={() => onSelect(article)}
        />
      ))}
    </scroll-view>
  )
}

function LinkList({ links, selectedID, onSelect }: { links: Link[], selectedID: string | null, onSelect: (link: Link) => void }) {
  if (links.length === 0) {
    return <StateBlock title='No links match these filters.' />
  }
  return (
    <scroll-view className='RecordList' scroll-y>
      {links.map(link => (
        <RecordItem
          key={link.id}
          active={link.id === selectedID}
          title={link.label}
          status={link.status}
          meta={`${link.icon_class || 'no icon'} - order ${link.sort_order}`}
          summary={link.url}
          pills={[link.star ? 'starred' : '', link.icon_class ?? '']}
          onTap={() => onSelect(link)}
        />
      ))}
    </scroll-view>
  )
}

function RecordItem(props: { active: boolean, title: string, status: Status, meta: string, summary: string, pills: string[], onTap: () => void }) {
  return (
    <view className={props.active ? 'RecordItem RecordItem--active' : 'RecordItem'} bindtap={props.onTap}>
      <view className='RecordHeader'>
        <text className='RecordTitle'>{props.title}</text>
        <text className={`StatusPill StatusPill--${props.status}`}>{props.status}</text>
      </view>
      <text className='RecordMeta'>{props.meta}</text>
      <text className='RecordSummary'>{props.summary}</text>
      <view className='TagRow'>
        {props.pills.filter(Boolean).map(pill => <text key={pill} className='MiniPill'>{pill}</text>)}
      </view>
    </view>
  )
}

function ProjectEditor(props: {
  form: ProjectForm
  mode: Mode
  selected: Project | null
  operation: Operation
  onFormChange: (form: ProjectForm) => void
  onSave: () => void
  onDelete: () => void
  onUploadMain: () => void
  onUploadGallery: () => void
  onDeleteImage: (imageID: string) => void
}) {
  const { form, mode, selected, operation, onFormChange, onSave, onDelete, onUploadMain, onUploadGallery, onDeleteImage } = props
  const update = (patch: Partial<ProjectForm>) => onFormChange({ ...form, ...patch })

  return (
    <view className='Editor'>
      <EditorHeader title={mode === 'create' ? 'Create project' : 'Edit project'} id={selected?.id} operation={operation} onSave={onSave} onDelete={onDelete} canDelete={!!selected} />
      <view className='FormGrid'>
        <TextField label='Title' value={form.title} onChange={title => update({ title })} />
        <TextField label='Slug' value={form.slug} placeholder='auto-generated if blank' onChange={slug => update({ slug })} />
        <TextField label='Summary' value={form.summary} onChange={summary => update({ summary })} multiline />
        <TextField label='Description' value={form.description} onChange={description => update({ description })} multiline />
        <TextField label='Body' value={form.body} onChange={body => update({ body })} multiline />
        <TextField label='Tech stack' value={form.techStack} placeholder='Go, PostgreSQL, React' onChange={techStack => update({ techStack })} />
        <TextField label='Tags' value={form.tags} placeholder='admin, backend' onChange={tags => update({ tags })} />
        <TextField label='GitHub URL' value={form.githubUrl} onChange={githubUrl => update({ githubUrl })} />
        <TextField label='Demo URL' value={form.demoUrl} onChange={demoUrl => update({ demoUrl })} />
        <TextField label='Sort order' value={form.sortOrder} numeric onChange={sortOrder => update({ sortOrder })} />
        <StatusField value={form.status} onChange={status => update({ status })} />
        <ToggleField label='Featured' value={form.featured} onLabel='Featured' offLabel='Standard' onChange={featured => update({ featured })} />
      </view>
      <ImagePanel
        title='Project images'
        disabled={!selected || operation !== 'idle'}
        altText={form.imageAltText}
        caption={form.imageCaption}
        primaryLabel='Upload main'
        secondaryLabel='Upload gallery'
        onAltTextChange={imageAltText => update({ imageAltText })}
        onCaptionChange={imageCaption => update({ imageCaption })}
        onUploadPrimary={onUploadMain}
        onUploadSecondary={onUploadGallery}
      />
      <view className='ImageGrid'>
        {selected?.main_image ? <ImageCard title='Main image' image={selected.main_image} onDelete={onDeleteImage} /> : <StateBlock title='No main image uploaded.' compact />}
        {(selected?.images ?? []).map(image => <ImageCard key={image.id} title='Gallery image' image={image} onDelete={onDeleteImage} />)}
      </view>
    </view>
  )
}

function CertificateEditor(props: {
  form: CertificateForm
  mode: Mode
  selected: Certificate | null
  operation: Operation
  onFormChange: (form: CertificateForm) => void
  onSave: () => void
  onDelete: () => void
  onUploadImage: () => void
  onDeleteImage: () => void
}) {
  const { form, mode, selected, operation, onFormChange, onSave, onDelete, onUploadImage, onDeleteImage } = props
  const update = (patch: Partial<CertificateForm>) => onFormChange({ ...form, ...patch })

  return (
    <view className='Editor'>
      <EditorHeader title={mode === 'create' ? 'Create certificate' : 'Edit certificate'} id={selected?.id} operation={operation} onSave={onSave} onDelete={onDelete} canDelete={!!selected} />
      <view className='FormGrid'>
        <TextField label='Title' value={form.title} onChange={title => update({ title })} />
        <TextField label='Issuer' value={form.issuer} onChange={issuer => update({ issuer })} />
        <TextField label='Slug' value={form.slug} placeholder='auto-generated if blank' onChange={slug => update({ slug })} />
        <TextField label='Summary' value={form.summary} onChange={summary => update({ summary })} multiline />
        <TextField label='Description' value={form.description} onChange={description => update({ description })} multiline />
        <TextField label='Credential URL' value={form.credentialUrl} onChange={credentialUrl => update({ credentialUrl })} />
        <TextField label='Issued at' value={form.issuedAt} placeholder='YYYY-MM-DD' onChange={issuedAt => update({ issuedAt })} />
        <TextField label='Expires at' value={form.expiresAt} placeholder='YYYY-MM-DD' onChange={expiresAt => update({ expiresAt })} />
        <TextField label='Sort order' value={form.sortOrder} numeric onChange={sortOrder => update({ sortOrder })} />
        <StatusField value={form.status} onChange={status => update({ status })} />
        <ToggleField label='Featured' value={form.featured} onLabel='Featured' offLabel='Standard' onChange={featured => update({ featured })} />
      </view>
      <ImagePanel
        title='Certificate image'
        disabled={!selected || operation !== 'idle'}
        altText={form.imageAltText}
        caption={form.imageCaption}
        primaryLabel='Upload image'
        onAltTextChange={imageAltText => update({ imageAltText })}
        onCaptionChange={imageCaption => update({ imageCaption })}
        onUploadPrimary={onUploadImage}
      />
      <view className='ImageGrid'>
        {selected?.image ? <ImageCard title='Certificate image' image={selected.image} onDelete={() => onDeleteImage()} /> : <StateBlock title='No certificate image uploaded.' compact />}
      </view>
    </view>
  )
}

function ArticleEditor(props: {
  form: ArticleForm
  mode: Mode
  selected: Article | null
  operation: Operation
  onFormChange: (form: ArticleForm) => void
  onSave: () => void
  onDelete: () => void
}) {
  const { form, mode, selected, operation, onFormChange, onSave, onDelete } = props
  const update = (patch: Partial<ArticleForm>) => onFormChange({ ...form, ...patch })

  return (
    <view className='Editor'>
      <EditorHeader title={mode === 'create' ? 'Create article' : 'Edit article'} id={selected?.id} operation={operation} onSave={onSave} onDelete={onDelete} canDelete={!!selected} />
      <view className='FormGrid'>
        <TextField label='Title' value={form.title} onChange={title => update({ title })} />
        <TextField label='Source' value={form.source} onChange={source => update({ source })} />
        <TextField label='URL' value={form.url} onChange={url => update({ url })} />
        <TextField label='Cover image URL' value={form.coverImageUrl} onChange={coverImageUrl => update({ coverImageUrl })} />
        <TextField label='Summary' value={form.summary} onChange={summary => update({ summary })} multiline />
        <TextField label='Published at' value={form.publishedAt} placeholder='YYYY-MM-DD' onChange={publishedAt => update({ publishedAt })} />
        <TextField label='Sort order' value={form.sortOrder} numeric onChange={sortOrder => update({ sortOrder })} />
        <StatusField value={form.status} onChange={status => update({ status })} />
        <ToggleField label='Featured' value={form.featured} onLabel='Featured' offLabel='Standard' onChange={featured => update({ featured })} />
      </view>
    </view>
  )
}

function LinkEditor(props: {
  form: LinkForm
  mode: Mode
  selected: Link | null
  operation: Operation
  onFormChange: (form: LinkForm) => void
  onSave: () => void
  onDelete: () => void
}) {
  const { form, mode, selected, operation, onFormChange, onSave, onDelete } = props
  const update = (patch: Partial<LinkForm>) => onFormChange({ ...form, ...patch })

  return (
    <view className='Editor'>
      <EditorHeader title={mode === 'create' ? 'Create link' : 'Edit link'} id={selected?.id} operation={operation} onSave={onSave} onDelete={onDelete} canDelete={!!selected} />
      <view className='FormGrid'>
        <TextField label='Label' value={form.label} onChange={label => update({ label })} />
        <TextField label='URL' value={form.url} onChange={url => update({ url })} />
        <TextField label='Icon class' value={form.iconClass} placeholder='fa-brands fa-github' onChange={iconClass => update({ iconClass })} />
        <TextField label='Sort order' value={form.sortOrder} numeric onChange={sortOrder => update({ sortOrder })} />
        <StatusField value={form.status} onChange={status => update({ status })} />
        <ToggleField label='Star' value={form.star} onLabel='Starred' offLabel='Standard' onChange={star => update({ star })} />
      </view>
    </view>
  )
}

function IntroEditor(props: {
  form: IntroForm
  intro: Intro | null
  operation: Operation
  onFormChange: (form: IntroForm) => void
  onSave: () => void
  onDelete: () => void
  onUploadPicture: () => void
  onDeletePicture: () => void
}) {
  const { form, intro, operation, onFormChange, onSave, onDelete, onUploadPicture, onDeletePicture } = props
  const update = (patch: Partial<IntroForm>) => onFormChange({ ...form, ...patch })

  return (
    <view className='Editor'>
      <EditorHeader title='Edit intro' id={intro?.id ?? 'default'} operation={operation} onSave={onSave} onDelete={onDelete} canDelete={!!intro} />
      <view className='FormGrid'>
        <TextField label='Title' value={form.title} onChange={title => update({ title })} />
        <TextField label='Description' value={form.description} onChange={description => update({ description })} multiline />
      </view>
      <ImagePanel
        title='Profile picture'
        disabled={operation !== 'idle'}
        altText={form.imageAltText}
        caption={form.imageCaption}
        primaryLabel='Upload picture'
        onAltTextChange={imageAltText => update({ imageAltText })}
        onCaptionChange={imageCaption => update({ imageCaption })}
        onUploadPrimary={onUploadPicture}
      />
      <view className='ImageGrid'>
        {intro?.profile_picture ? <ImageCard title='Profile picture' image={intro.profile_picture} onDelete={() => onDeletePicture()} /> : <StateBlock title='No profile picture uploaded.' compact />}
      </view>
    </view>
  )
}

function EditorHeader({ title, id, operation, onSave, onDelete, canDelete }: { title: string, id?: string, operation: Operation, onSave: () => void, onDelete: () => void, canDelete: boolean }) {
  return (
    <view className='EditorHeader'>
      <view>
        <text className='EditorTitle'>{title}</text>
        <text className='MutedText'>{id || 'New draft record'}</text>
      </view>
      <view className='EditorActions'>
        <TapButton label={operation === 'saving' ? 'Saving...' : 'Save'} onTap={onSave} variant='primary' disabled={operation !== 'idle'} />
        <TapButton label='Delete' onTap={onDelete} variant='danger' disabled={!canDelete || operation !== 'idle'} />
      </view>
    </view>
  )
}

function StatusField({ value, onChange }: { value: Status, onChange: (status: Status) => void }) {
  return <SelectGroup label='Status' value={value} values={EDIT_STATUS_OPTIONS} onChange={onChange} />
}

interface ImagePanelProps {
  title: string
  disabled: boolean
  altText: string
  caption: string
  primaryLabel: string
  secondaryLabel?: string
  onAltTextChange: (value: string) => void
  onCaptionChange: (value: string) => void
  onUploadPrimary: () => void
  onUploadSecondary?: () => void
}

function ImagePanel(props: ImagePanelProps) {
  return (
    <view className='UploadPanel'>
      <view>
        <text className='SectionTitle'>{props.title}</text>
        <text className='MutedText'>PNG or JPG, sent as multipart/form-data.</text>
      </view>
      <view className='FormGrid FormGrid--compact'>
        <TextField label='Alt text' value={props.altText} onChange={props.onAltTextChange} />
        <TextField label='Caption' value={props.caption} onChange={props.onCaptionChange} />
      </view>
      <view className='InlineActions'>
        <TapButton label={props.primaryLabel} onTap={props.onUploadPrimary} variant='secondary' disabled={props.disabled} />
        {props.secondaryLabel && props.onUploadSecondary ? <TapButton label={props.secondaryLabel} onTap={props.onUploadSecondary} variant='secondary' disabled={props.disabled} /> : null}
      </view>
    </view>
  )
}

function ImageCard({ title, image, onDelete }: { title: string, image: CommonImage, onDelete: (imageID: string) => void }) {
  return (
    <view className='ImageCard'>
      <view className='ImagePreview'>
        <image src={resolveAssetUrl(image.url)} className='PreviewImage' />
      </view>
      <view className='ImageMeta'>
        <text className='SectionTitle'>{title}</text>
        <text className='MutedText'>{image.alt_text || 'No alt text'} - {image.width || 0}x{image.height || 0}</text>
        <text className='MutedText'>{image.content_type || 'image'} - {formatBytes(image.size_bytes)}</text>
      </view>
      <TapButton label='Delete image' onTap={() => onDelete(image.id)} variant='danger' />
    </view>
  )
}

function IntroSummary({ intro, loading }: { intro: Intro | null, loading: boolean }) {
  if (loading) {
    return <view className='Filters'><StateBlock title='Loading intro...' compact /></view>
  }
  return (
    <view className='Filters'>
      <text className='SectionTitle'>{intro?.title || 'No intro content'}</text>
      <text className='RecordSummary'>{intro?.description || 'Save intro content to create the singleton record.'}</text>
      {intro?.profile_picture ? <text className='MiniPill'>profile picture</text> : null}
    </view>
  )
}

function ErrorPanel({ error, fields }: { error: string, fields: ApiFieldErrors }) {
  const entries = Object.entries(fields)
  return (
    <view className='ErrorPanel'>
      <text className='ErrorTitle'>{error}</text>
      {entries.map(([field, message]) => <text key={field} className='ErrorDetail'>{field}: {message}</text>)}
    </view>
  )
}

function StateBlock({ title, compact = false }: { title: string, compact?: boolean }) {
  return (
    <view className={compact ? 'StateBlock StateBlock--compact' : 'StateBlock'}>
      <text className='MutedText'>{title}</text>
    </view>
  )
}

function contentFilterFor(tab: ResourceTab, projectFilter: ListFilter, certificateFilter: ListFilter, articleFilter: ListFilter): ListFilter {
  if (tab === 'certificates') return certificateFilter
  if (tab === 'articles') return articleFilter
  return projectFilter
}

function newButtonLabel(tab: ResourceTab): string {
  switch (tab) {
    case 'certificates':
      return 'New certificate'
    case 'articles':
      return 'New article'
    case 'links':
      return 'New link'
    default:
      return 'New project'
  }
}

function newActiveRecord(tab: ResourceTab, actions: {
  setSelectedProjectID: (id: string | null) => void
  setProjectForm: (form: ProjectForm) => void
  setProjectMode: (mode: Mode) => void
  setSelectedCertificateID: (id: string | null) => void
  setCertificateForm: (form: CertificateForm) => void
  setCertificateMode: (mode: Mode) => void
  setSelectedArticleID: (id: string | null) => void
  setArticleForm: (form: ArticleForm) => void
  setArticleMode: (mode: Mode) => void
  setSelectedLinkID: (id: string | null) => void
  setLinkForm: (form: LinkForm) => void
  setLinkMode: (mode: Mode) => void
  clearError: () => void
}) {
  if (tab === 'projects') newProjectState(actions.setSelectedProjectID, actions.setProjectForm, actions.setProjectMode)
  if (tab === 'certificates') newCertificateState(actions.setSelectedCertificateID, actions.setCertificateForm, actions.setCertificateMode)
  if (tab === 'articles') newArticleState(actions.setSelectedArticleID, actions.setArticleForm, actions.setArticleMode)
  if (tab === 'links') newLinkState(actions.setSelectedLinkID, actions.setLinkForm, actions.setLinkMode)
  actions.clearError()
}

function selectProjectState(project: Project, setID: (id: string | null) => void, setForm: (form: ProjectForm) => void, setMode: (mode: Mode) => void) {
  setID(project.id)
  setForm(projectToForm(project))
  setMode('edit')
}

function selectCertificateState(certificate: Certificate, setID: (id: string | null) => void, setForm: (form: CertificateForm) => void, setMode: (mode: Mode) => void) {
  setID(certificate.id)
  setForm(certificateToForm(certificate))
  setMode('edit')
}

function selectArticleState(article: Article, setID: (id: string | null) => void, setForm: (form: ArticleForm) => void, setMode: (mode: Mode) => void) {
  setID(article.id)
  setForm(articleToForm(article))
  setMode('edit')
}

function selectLinkState(link: Link, setID: (id: string | null) => void, setForm: (form: LinkForm) => void, setMode: (mode: Mode) => void) {
  setID(link.id)
  setForm(linkToForm(link))
  setMode('edit')
}

function newProjectState(setID: (id: string | null) => void, setForm: (form: ProjectForm) => void, setMode: (mode: Mode) => void) {
  setID(null)
  setForm(emptyProjectForm)
  setMode('create')
}

function newCertificateState(setID: (id: string | null) => void, setForm: (form: CertificateForm) => void, setMode: (mode: Mode) => void) {
  setID(null)
  setForm(emptyCertificateForm)
  setMode('create')
}

function newArticleState(setID: (id: string | null) => void, setForm: (form: ArticleForm) => void, setMode: (mode: Mode) => void) {
  setID(null)
  setForm(emptyArticleForm)
  setMode('create')
}

function newLinkState(setID: (id: string | null) => void, setForm: (form: LinkForm) => void, setMode: (mode: Mode) => void) {
  setID(null)
  setForm(emptyLinkForm)
  setMode('create')
}

function readToken(): string {
  try {
    return storage()?.getItem(TOKEN_KEY) ?? ''
  } catch {
    return ''
  }
}

function writeToken(token: string): void {
  try {
    const activeStorage = storage()
    if (!activeStorage) return
    if (token) activeStorage.setItem(TOKEN_KEY, token)
    else activeStorage.removeItem(TOKEN_KEY)
  } catch {
    // Storage can be unavailable in non-web runtimes.
  }
}

function storage(): Storage | undefined {
  return (globalThis as typeof globalThis & { localStorage?: Storage }).localStorage
}

function handleError(nextError: unknown, setError: (message: string) => void, setFieldErrors: (fields: ApiFieldErrors) => void) {
  if (nextError instanceof ApiError) {
    setError(nextError.message)
    setFieldErrors(nextError.fields ?? {})
    return
  }
  if (nextError instanceof Error && nextError.name === 'AbortError') return
  if (nextError instanceof Error) {
    setError(nextError.message)
    setFieldErrors({})
    return
  }
  setError('Request failed')
  setFieldErrors({})
}

function eventValue(event: unknown): string {
  const inputEvent = event as {
    detail?: { value?: string }
    value?: string
    target?: { value?: string }
  }
  return inputEvent.detail?.value ?? inputEvent.value ?? inputEvent.target?.value ?? ''
}

function setElementValue(id: string, value: string): void {
  const lynxRef = (globalThis as typeof globalThis & {
    lynx?: {
      createSelectorQuery?: () => {
        select: (selector: string) => {
          invoke: (options: { method: string, params: { value: string } }) => { exec: () => void }
        }
      }
    }
  }).lynx

  try {
    lynxRef?.createSelectorQuery?.()
      .select(`#${id}`)
      .invoke({ method: 'setValue', params: { value } })
      .exec()
    return
  } catch {
    // Fall through to the web DOM path.
  }

  const documentRef = (globalThis as typeof globalThis & { document?: Document }).document
  const element = documentRef?.getElementById(id) as HTMLInputElement | HTMLTextAreaElement | null
  if (element && element.value !== value) {
    element.value = value
  }
}

function projectToForm(project: Project): ProjectForm {
  return {
    slug: project.slug,
    title: project.title,
    summary: project.summary ?? '',
    description: project.description ?? '',
    body: project.body ?? '',
    techStack: (project.tech_stack ?? []).join(', '),
    tags: (project.tags ?? []).join(', '),
    githubUrl: project.github_url ?? '',
    demoUrl: project.demo_url ?? '',
    featured: project.featured,
    sortOrder: String(project.sort_order),
    status: project.status,
    imageAltText: project.main_image?.alt_text ?? '',
    imageCaption: project.main_image?.caption ?? '',
  }
}

function certificateToForm(certificate: Certificate): CertificateForm {
  return {
    slug: certificate.slug,
    title: certificate.title,
    issuer: certificate.issuer,
    summary: certificate.summary ?? '',
    description: certificate.description ?? '',
    credentialUrl: certificate.credential_url ?? '',
    featured: certificate.featured,
    sortOrder: String(certificate.sort_order),
    status: certificate.status,
    issuedAt: dateOnly(certificate.issued_at),
    expiresAt: dateOnly(certificate.expires_at),
    imageAltText: certificate.image?.alt_text ?? '',
    imageCaption: certificate.image?.caption ?? '',
  }
}

function articleToForm(article: Article): ArticleForm {
  return {
    title: article.title,
    url: article.url,
    source: article.source,
    summary: article.summary ?? '',
    coverImageUrl: article.cover_image_url ?? '',
    featured: article.featured,
    sortOrder: String(article.sort_order),
    status: article.status,
    publishedAt: dateOnly(article.published_at),
  }
}

function linkToForm(link: Link): LinkForm {
  return {
    label: link.label,
    url: link.url,
    iconClass: link.icon_class ?? '',
    sortOrder: String(link.sort_order),
    star: link.star,
    status: link.status,
  }
}

function introToForm(intro: Intro): IntroForm {
  return {
    title: intro.title,
    description: intro.description,
    imageAltText: intro.profile_picture?.alt_text ?? '',
    imageCaption: intro.profile_picture?.caption ?? '',
  }
}

function projectPayload(form: ProjectForm): ProjectPayload {
  return {
    slug: form.slug.trim(),
    title: form.title.trim(),
    summary: form.summary.trim(),
    description: form.description.trim(),
    body: form.body.trim(),
    tech_stack: splitList(form.techStack),
    tags: splitList(form.tags),
    github_url: form.githubUrl.trim(),
    demo_url: form.demoUrl.trim(),
    featured: form.featured,
    sort_order: toInteger(form.sortOrder),
    status: form.status,
  }
}

function certificatePayload(form: CertificateForm): CertificatePayload {
  return {
    slug: form.slug.trim(),
    title: form.title.trim(),
    issuer: form.issuer.trim(),
    summary: form.summary.trim(),
    description: form.description.trim(),
    credential_url: form.credentialUrl.trim(),
    featured: form.featured,
    sort_order: toInteger(form.sortOrder),
    status: form.status,
    issued_at: dateValue(form.issuedAt),
    expires_at: dateValue(form.expiresAt),
  }
}

function articlePayload(form: ArticleForm): ArticlePayload {
  return {
    title: form.title.trim(),
    url: form.url.trim(),
    source: form.source.trim(),
    summary: form.summary.trim(),
    cover_image_url: form.coverImageUrl.trim(),
    featured: form.featured,
    sort_order: toInteger(form.sortOrder),
    status: form.status,
    published_at: dateValue(form.publishedAt),
  }
}

function linkPayload(form: LinkForm): LinkPayload {
  return {
    label: form.label.trim(),
    url: form.url.trim(),
    icon_class: form.iconClass.trim(),
    sort_order: toInteger(form.sortOrder),
    star: form.star,
    status: form.status,
  }
}

function introPayload(form: IntroForm): IntroPayload {
  return {
    title: form.title.trim(),
    description: form.description.trim(),
  }
}

function splitList(value: string): string[] {
  return value.split(',').map(item => item.trim()).filter(Boolean)
}

function toInteger(value: string): number {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : 0
}

function dateValue(value: string): string | null {
  const trimmed = value.trim()
  return trimmed ? `${trimmed}T00:00:00.000Z` : null
}

function dateOnly(value?: string): string {
  return value ? value.slice(0, 10) : ''
}

function labelize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

function formatBytes(value?: number): string {
  if (!value) return '0 B'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`
  return `${Math.round(value / 1024 / 1024)} MB`
}

async function optionalResult<T>(promise: Promise<T>, fallback: T): Promise<T> {
  try {
    return await promise
  } catch {
    return fallback
  }
}

function byFeaturedThenProjectOrder(first: Project, second: Project): number {
  if (first.featured !== second.featured) {
    return first.featured ? -1 : 1
  }
  return first.sort_order - second.sort_order || first.title.localeCompare(second.title)
}

function dateRange(startedAt?: string, endedAt?: string, current?: boolean): string {
  const start = dateOnly(startedAt)
  if (current) {
    return start ? `${start} - Present` : 'Current'
  }
  const end = dateOnly(endedAt)
  if (start && end) return `${start} - ${end}`
  if (start) return start
  return 'Published role'
}

function readInitialRoute(): AppRoute {
  const windowRef = browserWindow()
  if (!windowRef) return 'public'

  const url = new URL(windowRef.location.href)
  if (url.searchParams.get('route') === 'admin') {
    return 'admin'
  }

  const pathname = windowRef.location.pathname.replace(/\/+$/, '') || '/'
  return pathname === '/login' || pathname === '/admin' ? 'admin' : 'public'
}

function pushPath(path: '/' | '/login') {
  const windowRef = browserWindow()
  if (!windowRef) return

  let nextPath = path
  if (windowRef.location.pathname.includes('__web_preview')) {
    const url = new URL(windowRef.location.href)
    if (path === '/login') {
      url.searchParams.set('route', 'admin')
    } else {
      url.searchParams.delete('route')
    }
    nextPath = `${url.pathname}${url.search}${url.hash}` as '/' | '/login'
  }

  windowRef.history?.pushState?.(null, '', nextPath)
}

function browserWindow(): Window | undefined {
  return (globalThis as typeof globalThis & { window?: Window }).window
}

function injectOnestFontLinks(): void {
  const documentRef = browserWindow()?.document
  if (!documentRef?.head) return

  const links: Array<{ id: string, rel: string, href: string, crossorigin?: string }> = [
    {
      id: 'font-preconnect-googleapis',
      rel: 'preconnect',
      href: 'https://fonts.googleapis.com',
    },
    {
      id: 'font-preconnect-gstatic',
      rel: 'preconnect',
      href: 'https://fonts.gstatic.com',
      crossorigin: '',
    },
    {
      id: 'font-onest-stylesheet',
      rel: 'stylesheet',
      href: 'https://fonts.googleapis.com/css2?family=Onest:wght@100..900&display=swap',
    },
  ]

  for (const item of links) {
    if (documentRef.getElementById(item.id)) continue

    const link = documentRef.createElement('link')
    link.id = item.id
    link.rel = item.rel
    link.href = item.href
    if (item.crossorigin !== undefined) {
      link.setAttribute('crossorigin', item.crossorigin)
    }
    documentRef.head.appendChild(link)
  }
}

function openExternal(url: string): void {
  const trimmed = url.trim()
  if (!trimmed) return

  const windowRef = browserWindow()
  if (windowRef?.open) {
    windowRef.open(trimmed, '_blank', 'noopener,noreferrer')
    return
  }
  if (windowRef?.location) {
    windowRef.location.href = trimmed
  }
}

function canRenderPublicImage(url?: string): boolean {
  if (!url) return false
  if (url.startsWith('data:')) return true
  return /^https?:\/\//i.test(url)
}

function confirmAction(message: string): boolean {
  const confirmDialog = (globalThis as typeof globalThis & { confirm?: (message?: string) => boolean }).confirm
  return confirmDialog ? confirmDialog(message) : true
}

function pickImageFile(): Promise<File> {
  const documentRef = (globalThis as typeof globalThis & { document?: Document }).document
  if (!documentRef) {
    return Promise.reject(new Error('Image upload is available in the web client.'))
  }

  return new Promise((resolve, reject) => {
    const input = documentRef.createElement('input')
    input.type = 'file'
    input.accept = 'image/png,image/jpeg'
    input.onchange = () => {
      const file = input.files?.[0]
      if (file) resolve(file)
      else reject(new Error('No image selected'))
      input.remove()
    }
    input.click()
  })
}
