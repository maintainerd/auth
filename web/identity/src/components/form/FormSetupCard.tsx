import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

interface FormSetupCardProps {
  title: string
  description: string
  children: React.ReactNode
  className?: string
  headerClassName?: string
  contentClassName?: string
}

export default function FormSetupCard({
  title,
  description,
  children,
  className,
  headerClassName,
  contentClassName
}: FormSetupCardProps) {
  return (
    <Card className={`w-full max-w-md mx-auto ${className || ''}`}>
      <CardHeader className={`text-center ${headerClassName || ''}`}>
        <CardTitle className="text-xl">{title}</CardTitle>
        <CardDescription>
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent className={`px-8 ${contentClassName || ''}`}>
        {children}
      </CardContent>
    </Card>
  )
}
