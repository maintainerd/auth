interface MaintainedAuthIconProps {
  width?: number | string
  height?: number | string
  className?: string
  alt?: string
}

const MaintainedAuthIcon = ({
  width = 24,
  height = 24,
  className = '',
  alt = 'Maintainerd Auth logo',
}: MaintainedAuthIconProps) => {
  return (
    <img
      src="/logo.png"
      alt={alt}
      width={width}
      height={height}
      className={`object-contain ${className}`.trim()}
    />
  )
}

export default MaintainedAuthIcon
