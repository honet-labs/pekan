import React from 'react';

interface SkeletonProps {
  className?: string;
  width?: string | number;
  height?: string | number;
  circle?: boolean;
}

const Skeleton: React.FC<SkeletonProps> = ({ className = '', width, height, circle }) => {
  const style: React.CSSProperties = {
    width: width,
    height: height,
    borderRadius: circle ? '50%' : '4px',
  };

  return (
    <div 
      className={`skeleton-animate bg-gray-200 dark:bg-gray-700 ${className}`}
      style={style}
    />
  );
};

export default Skeleton;
