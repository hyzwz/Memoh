export interface robot{
  description: string
  time: Date,
  id: string | number,
  type: string,
  action:'robot'
}

export interface user{
  description: string, 
  time: Date, 
  id: number | string,
  action:'user'
}