interface MaintainedAuthIconProps {
  width?: number | string;
  height?: number | string;
  className?: string;
}

const MaintainedAuthIcon = ({
  width = 24,
  height = 24,
  className = ""
}: MaintainedAuthIconProps) => {
	return (
		<svg
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 24 24"
			width={width}
			height={height}
			className={className}
			role="img"
			aria-label="Maintainerd IAM dual-color shield"
		>
			<defs>
				<clipPath id="left-half">
					<rect x="0" y="0" width="12" height="24" />
				</clipPath>
				<clipPath id="right-half">
					<rect x="12" y="0" width="12" height="24" />
				</clipPath>
				<path id="shieldPath"
					d="M12 2.5
						L4 5.5
						V12
						C4 16.5 8 20.5 12 21.5
						C16 20.5 20 16.5 20 12
						V5.5
						L12 2.5Z" />
			</defs>
			<rect width="100%" height="100%" fill="transparent" />
			<g clipPath="url(#left-half)">
				<use href="#shieldPath" fill="#1E40AF" />
			</g>
			<g clipPath="url(#right-half)">
				<use href="#shieldPath" fill="#2563EB" />
			</g>
		</svg>

	)
}

export default MaintainedAuthIcon;
