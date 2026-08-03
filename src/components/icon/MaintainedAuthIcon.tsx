interface MaintainedAuthIconProps {
  width?: number | string
  height?: number | string
  className?: string
}

const MaintainedAuthIcon = ({
  width = 24,
  height = 24,
  className = '',
}: MaintainedAuthIconProps) => {
  return (
    <img
      src="/logo.png"
      alt="Maintainerd Auth logo"
      width={width}
      height={height}
      className={`object-contain ${className}`.trim()}
    />
  )
}

export default MaintainedAuthIcon
