import { cn } from '@/lib/utils'
import { useLanguage } from '../context/LanguageContext'

// Empty component
export default function Empty() {
  const { t } = useLanguage()
  return (
    <div className={cn('flex h-full items-center justify-center')}>{t('common.empty')}</div>
  )
}
