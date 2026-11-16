import type { GradingScale } from '@/types/course'

export const DEFAULT_HANDIN_MAX_POINTS = 5

export const DEFAULT_GRADING_SCALE: Readonly<GradingScale> = {
  scaleA: 90,
  scaleB: 80,
  scaleC: 60,
  scaleD: 50,
  scaleE: 40,
}

export function isValidGradingScale(scale: GradingScale): boolean {
  const values = [scale.scaleA, scale.scaleB, scale.scaleC, scale.scaleD, scale.scaleE]
  if (values.some(value => typeof value !== 'number' || !Number.isFinite(value))) {
    return false
  }
  return (
    scale.scaleA <= 100 &&
    scale.scaleA > scale.scaleB &&
    scale.scaleB > scale.scaleC &&
    scale.scaleC > scale.scaleD &&
    scale.scaleD > scale.scaleE &&
    scale.scaleE > 0
  )
}

export function getGradeBadgeColor(letter: string): string {
  switch (letter.toUpperCase()) {
    case 'A':
    case 'B': {
      return 'success'
    }
    case 'C':
    case 'D': {
      return 'warning'
    }
    case 'E':
    case 'F': {
      return 'error'
    }
    default: {
      return 'secondary'
    }
  }
}
